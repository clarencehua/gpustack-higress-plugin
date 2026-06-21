package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestParseConfig(t *testing.T) {
	makeConfig := func(v map[string]interface{}) gjson.Result {
		data, _ := json.Marshal(v)
		return gjson.ParseBytes(data)
	}

	t.Run("default values", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body":    map[string]interface{}{"enabled": true},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body":    map[string]interface{}{"enabled": true},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)

		require.True(t, cfg.Request.Headers.Enabled)
		require.True(t, cfg.Request.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Request.Body.MaxSize)
		require.Equal(t, []string{"application/json", "application/xml", "application/x-www-form-urlencoded", "text/plain"}, cfg.Request.Body.ContentTypes)

		require.True(t, cfg.Response.Headers.Enabled)
		require.True(t, cfg.Response.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Response.Body.MaxSize)
		require.Equal(t, []string{"application/json", "application/xml", "text/plain", "text/html"}, cfg.Response.Body.ContentTypes)
	})

	t.Run("custom values", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": false},
				"body": map[string]interface{}{
					"enabled":      true,
					"maxSize":      2048,
					"contentTypes": []string{"application/json", "text/plain"},
				},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled":      false,
					"maxSize":      4096,
					"contentTypes": []string{"text/html"},
				},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)

		require.False(t, cfg.Request.Headers.Enabled)
		require.True(t, cfg.Request.Body.Enabled)
		require.Equal(t, 2048, cfg.Request.Body.MaxSize)
		require.Equal(t, []string{"application/json", "text/plain"}, cfg.Request.Body.ContentTypes)

		require.True(t, cfg.Response.Headers.Enabled)
		require.False(t, cfg.Response.Body.Enabled)
		require.Equal(t, 4096, cfg.Response.Body.MaxSize)
	})

	t.Run("minimal config - all disabled", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": false},
				"body":    map[string]interface{}{"enabled": false},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": false},
				"body":    map[string]interface{}{"enabled": false},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)
		require.False(t, cfg.Request.Headers.Enabled)
		require.False(t, cfg.Request.Body.Enabled)
		require.False(t, cfg.Response.Headers.Enabled)
		require.False(t, cfg.Response.Body.Enabled)
	})

	t.Run("maxSize zero uses default", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled": true,
					"maxSize": 0,
				},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled": true,
					"maxSize": 0,
				},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)
		require.Equal(t, 10*1024, cfg.Request.Body.MaxSize)
		require.Equal(t, 10*1024, cfg.Response.Body.MaxSize)
	})

	t.Run("empty JSON uses all defaults", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)

		require.False(t, cfg.Request.Headers.Enabled)
		require.False(t, cfg.Request.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Request.Body.MaxSize)
		require.Equal(t, []string{"application/json", "application/xml", "application/x-www-form-urlencoded", "text/plain"}, cfg.Request.Body.ContentTypes)

		require.False(t, cfg.Response.Headers.Enabled)
		require.False(t, cfg.Response.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Response.Body.MaxSize)
		require.Equal(t, []string{"application/json", "application/xml", "text/plain", "text/html"}, cfg.Response.Body.ContentTypes)
	})

	t.Run("only request configured", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled": true,
					"maxSize": 512,
				},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)

		require.True(t, cfg.Request.Headers.Enabled)
		require.True(t, cfg.Request.Body.Enabled)
		require.Equal(t, 512, cfg.Request.Body.MaxSize)

		require.False(t, cfg.Response.Headers.Enabled)
		require.False(t, cfg.Response.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Response.Body.MaxSize)
	})

	t.Run("only response configured", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled":      true,
					"maxSize":      256,
					"contentTypes": []string{"text/plain"},
				},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)

		require.False(t, cfg.Request.Headers.Enabled)
		require.False(t, cfg.Request.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Request.Body.MaxSize)

		require.True(t, cfg.Response.Headers.Enabled)
		require.True(t, cfg.Response.Body.Enabled)
		require.Equal(t, 256, cfg.Response.Body.MaxSize)
		require.Equal(t, []string{"text/plain"}, cfg.Response.Body.ContentTypes)
	})

	t.Run("negative maxSize uses default", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled": true,
					"maxSize": -100,
				},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled": true,
					"maxSize": -1,
				},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)
		require.Equal(t, 10*1024, cfg.Request.Body.MaxSize)
		require.Equal(t, 10*1024, cfg.Response.Body.MaxSize)
	})

	t.Run("empty contentTypes array falls back to defaults (gjson treats [] as not set)", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled":      true,
					"contentTypes": []string{},
				},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
				"body": map[string]interface{}{
					"enabled":      true,
					"contentTypes": []string{},
				},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)
		// gjson's .Array() returns empty for [], len check falls back to defaults
		require.Equal(t, []string{"application/json", "application/xml", "application/x-www-form-urlencoded", "text/plain"}, cfg.Request.Body.ContentTypes)
		require.Equal(t, []string{"application/json", "application/xml", "text/plain", "text/html"}, cfg.Response.Body.ContentTypes)
	})

	t.Run("partial nested fields missing", func(t *testing.T) {
		raw := makeConfig(map[string]interface{}{
			"request": map[string]interface{}{
				"body": map[string]interface{}{"enabled": true},
			},
			"response": map[string]interface{}{
				"headers": map[string]interface{}{"enabled": true},
			},
		})
		var cfg PluginConfig
		err := parseConfig(raw, &cfg)
		require.NoError(t, err)

		require.False(t, cfg.Request.Headers.Enabled)
		require.True(t, cfg.Request.Body.Enabled)
		require.Equal(t, 10*1024, cfg.Request.Body.MaxSize)

		require.True(t, cfg.Response.Headers.Enabled)
		require.False(t, cfg.Response.Body.Enabled)
	})
}

func TestNormalizeHeaderName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{":authority", "authority"},
		{":method", "method"},
		{":path", "path"},
		{":scheme", "scheme"},
		{":status", "status"},
		{":custom-header", "custom-header"},
		{":x-forwarded-for", "x-forwarded-for"},
		{":x-real-ip", "x-real-ip"},
		{"content-type", "content-type"},
		{"content-length", "content-length"},
		{"x-forwarded-for", "x-forwarded-for"},
		{"user-agent", "user-agent"},
		{"authorization", "authorization"},
		{"accept", "accept"},
		{"host", "host"},
		{"", ""},
	}

	for _, c := range cases {
		got := normalizeHeaderName(c.input)
		if got != c.expected {
			t.Errorf("normalizeHeaderName(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestProcessBodyChunk(t *testing.T) {
	t.Run("disabled when maxSize is zero", func(t *testing.T) {
		result := processBodyChunk(nil, 0, []byte("hello"), false)
		require.Equal(t, []byte(nil), result.buffer)
		require.False(t, result.shouldLog)
	})

	t.Run("disabled when maxSize is negative", func(t *testing.T) {
		result := processBodyChunk(nil, -1, []byte("hello"), false)
		require.Equal(t, []byte(nil), result.buffer)
		require.False(t, result.shouldLog)
	})

	t.Run("small data fits in buffer, no end of stream", func(t *testing.T) {
		result := processBodyChunk(nil, 100, []byte("hello"), false)
		require.Equal(t, []byte("hello"), result.buffer)
		require.False(t, result.shouldLog)
		require.Empty(t, result.logValue)
	})

	t.Run("multiple chunks accumulate", func(t *testing.T) {
		result := processBodyChunk([]byte("hello"), 100, []byte(" world"), false)
		require.Equal(t, []byte("hello world"), result.buffer)
		require.False(t, result.shouldLog)
	})

	t.Run("end of stream logs accumulated data", func(t *testing.T) {
		result := processBodyChunk([]byte("hello"), 100, nil, true)
		require.True(t, result.shouldLog)
		require.Equal(t, "hello", result.logValue)
	})

	t.Run("chunk exceeds remaining capacity logs and clears buffer", func(t *testing.T) {
		result := processBodyChunk([]byte("hel"), 5, []byte("lo-remainder"), false)
		require.True(t, result.shouldLog)
		require.Equal(t, "hello", result.logValue)
	})

	t.Run("buffer already full does nothing unless end of stream", func(t *testing.T) {
		buf := []byte("hello")
		result := processBodyChunk(buf, 5, []byte(" world"), false)
		require.Equal(t, []byte("hello"), result.buffer)
		require.False(t, result.shouldLog)
	})

	t.Run("buffer already full logs at end of stream", func(t *testing.T) {
		buf := []byte("hello")
		result := processBodyChunk(buf, 5, nil, true)
		require.True(t, result.shouldLog)
		require.Equal(t, "hello", result.logValue)
	})

	t.Run("chunk fits exactly with no room to spare then EOS", func(t *testing.T) {
		result := processBodyChunk([]byte("hel"), 5, []byte("lo"), true)
		require.True(t, result.shouldLog)
		require.Equal(t, "hello", result.logValue)
	})

	t.Run("chunk fills exactly to capacity without EOS does not log", func(t *testing.T) {
		result := processBodyChunk([]byte("hel"), 5, []byte("lo"), false)
		require.False(t, result.shouldLog)
		require.Equal(t, []byte("hello"), result.buffer)
	})

	t.Run("accumulate across multiple chunks then flush at EOS", func(t *testing.T) {
		var buf []byte
		var logHits []string

		chunks := [][]byte{[]byte("abc"), []byte("def"), []byte("ghi")}
		for i, ch := range chunks {
			isEnd := i == len(chunks)-1
			result := processBodyChunk(buf, 100, ch, isEnd)
			if result.shouldLog {
				logHits = append(logHits, result.logValue)
			} else {
				buf = result.buffer
			}
		}

		require.Equal(t, []string{"abcdefghi"}, logHits)
	})

	t.Run("large data triggers mid-stream log and flushes rest at EOS", func(t *testing.T) {
		result := processBodyChunk(nil, 10, []byte(strings.Repeat("a", 15)), false)
		require.True(t, result.shouldLog)
		require.Equal(t, "aaaaaaaaaa", result.logValue)

		result2 := processBodyChunk(nil, 10, []byte("bbbbb"), true)
		require.True(t, result2.shouldLog)
		require.Equal(t, "bbbbb", result2.logValue)
	})

	t.Run("empty chunk with EOS on empty buffer", func(t *testing.T) {
		result := processBodyChunk(nil, 100, nil, true)
		require.True(t, result.shouldLog)
		require.Equal(t, "", result.logValue)
	})

	t.Run("empty chunk with EOS on non-empty buffer", func(t *testing.T) {
		result := processBodyChunk([]byte("data"), 100, nil, true)
		require.True(t, result.shouldLog)
		require.Equal(t, "data", result.logValue)
	})

	t.Run("empty chunk without EOS does nothing", func(t *testing.T) {
		result := processBodyChunk([]byte("data"), 100, nil, false)
		require.Equal(t, []byte("data"), result.buffer)
		require.False(t, result.shouldLog)
	})

	t.Run("empty chunk without EOS on empty buffer", func(t *testing.T) {
		result := processBodyChunk(nil, 100, nil, false)
		require.Equal(t, []byte(nil), result.buffer)
		require.False(t, result.shouldLog)
	})

	t.Run("chunk exactly equals maxSize from empty buffer then EOS", func(t *testing.T) {
		result := processBodyChunk(nil, 5, []byte("hello"), true)
		require.True(t, result.shouldLog)
		require.Equal(t, "hello", result.logValue)
	})

	t.Run("chunk exactly equals maxSize from empty buffer not EOS", func(t *testing.T) {
		result := processBodyChunk(nil, 5, []byte("hello"), false)
		require.False(t, result.shouldLog)
		require.Equal(t, []byte("hello"), result.buffer)
	})

	t.Run("maxSize 1 boundary", func(t *testing.T) {
		result := processBodyChunk(nil, 1, []byte("a"), false)
		require.False(t, result.shouldLog)
		require.Equal(t, []byte("a"), result.buffer)
	})

	t.Run("maxSize 1 with EOS", func(t *testing.T) {
		result := processBodyChunk(nil, 1, []byte("a"), true)
		require.True(t, result.shouldLog)
		require.Equal(t, "a", result.logValue)
	})

	t.Run("maxSize 1 chunk exceeds", func(t *testing.T) {
		result := processBodyChunk(nil, 1, []byte("abc"), false)
		require.True(t, result.shouldLog)
		require.Equal(t, "a", result.logValue)
	})

	t.Run("buffer one byte below maxSize chunk exactly 1 byte not EOS", func(t *testing.T) {
		result := processBodyChunk([]byte("abcd"), 5, []byte("e"), false)
		require.False(t, result.shouldLog)
		require.Equal(t, []byte("abcde"), result.buffer)
	})

	t.Run("buffer one byte below maxSize chunk exactly 1 byte EOS", func(t *testing.T) {
		result := processBodyChunk([]byte("abcd"), 5, []byte("e"), true)
		require.True(t, result.shouldLog)
		require.Equal(t, "abcde", result.logValue)
	})

	t.Run("buffer one byte below maxSize chunk exceeds by 1", func(t *testing.T) {
		result := processBodyChunk([]byte("abcd"), 5, []byte("ef"), false)
		require.True(t, result.shouldLog)
		require.Equal(t, "abcde", result.logValue)
	})

	t.Run("multiple overflow cycles", func(t *testing.T) {
		var buf []byte
		var logHits []string

		// maxSize 3, send chunks of 2: [ab][cd][ef][gh] with EOS on last
		chunks := [][]byte{[]byte("ab"), []byte("cd"), []byte("ef"), []byte("gh")}
		for i, ch := range chunks {
			isEnd := i == len(chunks)-1
			result := processBodyChunk(buf, 3, ch, isEnd)
			if result.shouldLog {
				logHits = append(logHits, result.logValue)
				buf = nil // simulate ctx.SetContext(key, []byte{}) after log
			} else {
				buf = result.buffer
			}
		}

		// "ab" -> buffer="ab" (2<3, no log)
		// "cd" -> remaining=1, "c" fills to "abc", log "abc", buffer cleared
		// "ef" -> buffer="ef" (2<3, no log)
		// "gh" -> remaining=1, "g" fills to "efg", log "efg", buffer cleared
		require.Equal(t, []string{"abc", "efg"}, logHits)
	})

	t.Run("very large maxSize never triggers log", func(t *testing.T) {
		var buf []byte
		var logHits []string

		chunks := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d"), []byte("e")}
		for i, ch := range chunks {
			isEnd := i == len(chunks)-1
			result := processBodyChunk(buf, 1000000, ch, isEnd)
			if result.shouldLog {
				logHits = append(logHits, result.logValue)
			} else {
				buf = result.buffer
			}
		}

		require.Equal(t, []string{"abcde"}, logHits)
	})

	t.Run("single byte chunks accumulating to maxSize", func(t *testing.T) {
		var buf []byte
		var logHits []string

		for i := 0; i < 5; i++ {
			isEnd := i == 4
			result := processBodyChunk(buf, 5, []byte{byte('a' + i)}, isEnd)
			if result.shouldLog {
				logHits = append(logHits, result.logValue)
			} else {
				buf = result.buffer
			}
		}

		require.Equal(t, []string{"abcde"}, logHits)
	})

	t.Run("chunk exactly fills then more data arrives after clear", func(t *testing.T) {
		// maxSize 4, buffer "ab", chunk "cd" fills exactly -> no log, buffer="abcd"
		result := processBodyChunk([]byte("ab"), 4, []byte("cd"), false)
		require.False(t, result.shouldLog)
		require.Equal(t, []byte("abcd"), result.buffer)

		// next chunk "ef" overflows by 2 -> log "efgh" wait no, remaining=0
		// buffer="abcd" len=4 >= maxSize=4, so first check: len(buffer) >= maxSize
		// not EOS, so return buffer unchanged
		result2 := processBodyChunk(result.buffer, 4, []byte("ef"), false)
		require.False(t, result2.shouldLog)
		require.Equal(t, []byte("abcd"), result2.buffer)

		// EOS -> log "abcd"
		result3 := processBodyChunk(result2.buffer, 4, nil, true)
		require.True(t, result3.shouldLog)
		require.Equal(t, "abcd", result3.logValue)
	})
}
