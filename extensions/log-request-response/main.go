package main

import (
	"encoding/json"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	pluginName            = "log-request-response"
	logKeyRequestHeaders  = "log-request-headers"
	logKeyRequestBody     = "log-request-body"
	logKeyResponseHeaders = "log-response-headers"
	logKeyResponseBody    = "log-response-body"
)

const (
	contextKeyRequestBodyBuffer  = "request_body_buffer"
	contextKeyResponseBodyBuffer = "response_body_buffer"
)

var http2HeaderMap = map[string]string{
	":authority": "authority",
	":method":    "method",
	":path":      "path",
	":scheme":    "scheme",
	":status":    "status",
}

func main() {}

func init() {
	wrapper.SetCtx(
		pluginName,
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessStreamingRequestBody(onStreamingRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBody(onStreamingResponseBody),
	)
}

type PluginConfig struct {
	Request struct {
		Headers struct {
			Enabled bool
		}
		Body struct {
			Enabled      bool
			MaxSize      int
			ContentTypes []string
		}
	}
	Response struct {
		Headers struct {
			Enabled bool
		}
		Body struct {
			Enabled      bool
			MaxSize      int
			ContentTypes []string
		}
	}
}

func parseConfig(json gjson.Result, config *PluginConfig) error {
	if v := json.Get("request.headers.enabled"); v.Exists() {
		config.Request.Headers.Enabled = v.Bool()
	} else {
		config.Request.Headers.Enabled = true
	}

	if v := json.Get("request.body.enabled"); v.Exists() {
		config.Request.Body.Enabled = v.Bool()
	} else {
		config.Request.Body.Enabled = true
	}
	config.Request.Body.MaxSize = int(json.Get("request.body.maxSize").Int())

	if config.Request.Body.MaxSize <= 0 {
		config.Request.Body.MaxSize = 10 * 1024
	}

	if contentTypes := json.Get("request.body.contentTypes").Array(); len(contentTypes) > 0 {
		for _, ct := range contentTypes {
			config.Request.Body.ContentTypes = append(config.Request.Body.ContentTypes, ct.String())
		}
	} else {
		config.Request.Body.ContentTypes = []string{
			"application/json",
			"application/xml",
			"application/x-www-form-urlencoded",
			"text/plain",
		}
	}

	if v := json.Get("response.headers.enabled"); v.Exists() {
		config.Response.Headers.Enabled = v.Bool()
	} else {
		config.Response.Headers.Enabled = true
	}

	if v := json.Get("response.body.enabled"); v.Exists() {
		config.Response.Body.Enabled = v.Bool()
	} else {
		config.Response.Body.Enabled = true
	}
	config.Response.Body.MaxSize = int(json.Get("response.body.maxSize").Int())

	if config.Response.Body.MaxSize <= 0 {
		config.Response.Body.MaxSize = 10 * 1024
	}

	if contentTypes := json.Get("response.body.contentTypes").Array(); len(contentTypes) > 0 {
		for _, ct := range contentTypes {
			config.Response.Body.ContentTypes = append(config.Response.Body.ContentTypes, ct.String())
		}
	} else {
		config.Response.Body.ContentTypes = []string{
			"application/json",
			"application/xml",
			"text/plain",
			"text/html",
		}
	}

	return nil
}

func normalizeHeaderName(name string) string {
	if standardName, exists := http2HeaderMap[name]; exists {
		return standardName
	}

	if strings.HasPrefix(name, ":") {
		return name[1:]
	}

	return name
}

func setPropertyWithMarshal(key string, value string) {
	helper := map[string]string{
		"placeholder": value,
	}

	marshalledHelper, _ := json.Marshal(helper)
	marshalledRaw := gjson.GetBytes(marshalledHelper, "placeholder").Raw

	var marshalledStr string
	if len(marshalledRaw) >= 2 {
		marshalledStr = marshalledRaw[1 : len(marshalledRaw)-1]
	} else {
		log.Errorf("failed to marshal json string, raw string is: %s", value)
		marshalledStr = ""
	}

	if err := proxywasm.SetProperty([]string{key}, []byte(marshalledStr)); err != nil {
		log.Errorf("failed to set %s in filter state, err: %v, raw:\n%s", key, err, value)
	}
}

type bodyChunkResult struct {
	buffer    []byte
	logValue  string
	shouldLog bool
}

func processBodyChunk(buffer []byte, maxSize int, chunk []byte, isEndStream bool) bodyChunkResult {
	if maxSize <= 0 {
		return bodyChunkResult{buffer: buffer}
	}

	if len(buffer) >= maxSize {
		if isEndStream {
			return bodyChunkResult{logValue: string(buffer), shouldLog: true}
		}
		return bodyChunkResult{buffer: buffer}
	}

	remainingCapacity := maxSize - len(buffer)
	if remainingCapacity >= len(chunk) {
		buffer = append(buffer, chunk...)
		if isEndStream {
			return bodyChunkResult{logValue: string(buffer), shouldLog: true}
		}
		return bodyChunkResult{buffer: buffer}
	}

	buffer = append(buffer, chunk[:remainingCapacity]...)
	return bodyChunkResult{
		logValue:  string(buffer),
		shouldLog: true,
	}
}

func processStreamingBody(
	ctx wrapper.HttpContext,
	enabled bool,
	maxSize int,
	bufferKey string,
	logKey string,
	chunk []byte,
	isEndStream bool,
) []byte {
	if !enabled || maxSize <= 0 {
		return chunk
	}

	buffer, _ := ctx.GetContext(bufferKey).([]byte)
	result := processBodyChunk(buffer, maxSize, chunk, isEndStream)

	if result.shouldLog {
		setPropertyWithMarshal(logKey, result.logValue)
		ctx.SetContext(bufferKey, []byte{})
	} else {
		ctx.SetContext(bufferKey, result.buffer)
	}

	return chunk
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("Failed to get request headers: %v", err)
		return types.ActionContinue
	}

	method := ""
	contentType := ""

	if config.Request.Headers.Enabled {
		jsonStr := "{}"
		for _, header := range headers {
			var err error
			normalizedName := normalizeHeaderName(header[0])
			jsonStr, err = sjson.Set(jsonStr, normalizedName, header[1])
			if err != nil {
				log.Errorf("Failed to convert request header to JSON: name=%s, value=%s, error=%v", normalizedName, header[1], err)
			}
		}

		setPropertyWithMarshal(logKeyRequestHeaders, jsonStr)
	}

	for _, header := range headers {
		if strings.ToLower(header[0]) == ":method" {
			method = header[1]
		} else if strings.ToLower(header[0]) == "content-type" {
			contentType = header[1]
		}
	}

	if !config.Request.Body.Enabled || (method != "POST" && method != "PUT" && method != "PATCH") {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	shouldLogBody := false
	for _, allowedType := range config.Request.Body.ContentTypes {
		if strings.Contains(contentType, allowedType) {
			shouldLogBody = true
			break
		}
	}

	if !shouldLogBody {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	ctx.SetContext(contextKeyRequestBodyBuffer, []byte{})

	return types.ActionContinue
}

func onStreamingRequestBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool) []byte {
	return processStreamingBody(
		ctx,
		config.Request.Body.Enabled,
		config.Request.Body.MaxSize,
		contextKeyRequestBodyBuffer,
		logKeyRequestBody,
		chunk,
		isEndStream,
	)
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("Failed to get response headers: %v", err)
		return types.ActionContinue
	}

	if config.Response.Headers.Enabled {
		jsonStr := "{}"
		for _, header := range headers {
			var err error
			normalizedName := normalizeHeaderName(header[0])
			jsonStr, err = sjson.Set(jsonStr, normalizedName, header[1])
			if err != nil {
				log.Errorf("Failed to convert response header to JSON: name=%s, value=%s, error=%v", normalizedName, header[1], err)
			}
		}

		setPropertyWithMarshal(logKeyResponseHeaders, jsonStr)
	}

	if !config.Response.Body.Enabled {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	contentType := ""
	hasContentEncoding := false
	for _, header := range headers {
		if strings.ToLower(header[0]) == "content-type" {
			contentType = header[1]
		} else if strings.ToLower(header[0]) == "content-encoding" {
			hasContentEncoding = true
		}
	}

	if hasContentEncoding {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	if contentType != "" {
		shouldLogBody := false
		for _, allowedType := range config.Response.Body.ContentTypes {
			if strings.Contains(contentType, allowedType) {
				shouldLogBody = true
				break
			}
		}

		if !shouldLogBody {
			ctx.DontReadResponseBody()
			return types.ActionContinue
		}
	}

	ctx.SetContext(contextKeyResponseBodyBuffer, []byte{})

	return types.ActionContinue
}

func onStreamingResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool) []byte {
	return processStreamingBody(
		ctx,
		config.Response.Body.Enabled,
		config.Response.Body.MaxSize,
		contextKeyResponseBodyBuffer,
		logKeyResponseBody,
		chunk,
		isEndStream,
	)
}
