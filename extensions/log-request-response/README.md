# log-request-response

将请求/响应头和 body 写入 Higress 访问日志的 Wasm 插件。

## 在新 Higress 环境上的配置步骤

### 1. 修改访问日志格式

在 `higress-config` ConfigMap 的 **`higress`** 段落（不是 `mesh`）加上以下字段：

```yaml
data:
  higress: |-
    accessLogEncoding: TEXT
    accessLogFile: /dev/stdout
    accessLogFormat: '{"log_request_headers":"%FILTER_STATE(wasm.log-request-headers:PLAIN)%","log_request_body":"%FILTER_STATE(wasm.log-request-body:PLAIN)%","log_response_headers":"%FILTER_STATE(wasm.log-response-headers:PLAIN)%","log_response_body":"%FILTER_STATE(wasm.log-response-body:PLAIN)%","authority":"%REQ(X-ENVOY-ORIGINAL-HOST?:AUTHORITY)%","bytes_received":"%BYTES_RECEIVED%","bytes_sent":"%BYTES_SENT%","downstream_local_address":"%DOWNSTREAM_LOCAL_ADDRESS%","downstream_remote_address":"%DOWNSTREAM_REMOTE_ADDRESS%","duration":"%DURATION%","method":"%REQ(:METHOD)%","path":"%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%","protocol":"%PROTOCOL%","request_id":"%REQ(X-REQUEST-ID)%","response_code":"%RESPONSE_CODE%","response_flags":"%RESPONSE_FLAGS%","route_name":"%ROUTE_NAME%","start_time":"%START_TIME%","upstream_cluster":"%UPSTREAM_CLUSTER%","upstream_host":"%UPSTREAM_HOST%","upstream_service_time":"%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%","user_agent":"%REQ(USER-AGENT)%","x_forwarded_for":"%REQ(X-FORWARDED-FOR)%","response_code_details":"%RESPONSE_CODE_DETAILS%"}'
    downstream:
      ...
```

放在 **`higress`** 段落中修改无需重启 Gateway。

### 2. 上传并安装插件

在 Higress 控制台 → 插件管理 → 上传 `plugin.wasm` 和 `metadata.txt`，版本 `1.0.0`。

### 3. 创建 WasmPlugin

| 字段 | 值 |
|---|---|
| 执行阶段 | `UNSPECIFIED_PHASE` |
| 优先级 | `500` |
| 启用默认配置 | 关闭（`defaultConfigDisable: true`） |
| 默认配置 | 留空（不需要填） |
| URL | 插件服务器地址，如 `https://your-server/wasm-plugins/log-request-response/1.0.0/plugin.wasm` |

### 4. 在路由上启用

进入目标路由 → 插件标签页 → 点击 `log-request-response` 的**启用**按钮。**不需要填写任何配置 JSON**，所有日志字段默认开启。

如果某个路由上想禁用特定日志类型，可以在该路由的插件配置中指定：

```json
{
  "request": {
    "headers": { "enabled": false }
  }
}
```

## 配置字段说明

| 路径 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `request.headers.enabled` | bool | `true` | 是否记录请求头 |
| `request.body.enabled` | bool | `true` | 是否记录请求体 |
| `request.body.maxSize` | int | `10240` | 请求体最大字节数 |
| `request.body.contentTypes` | []string | `["application/json","application/xml","application/x-www-form-urlencoded","text/plain"]` | 记录请求体的 Content-Type 白名单 |
| `response.headers.enabled` | bool | `true` | 是否记录响应头 |
| `response.body.enabled` | bool | `true` | 是否记录响应体 |
| `response.body.maxSize` | int | `10240` | 响应体最大字节数 |
| `response.body.contentTypes` | []string | `["application/json","application/xml","text/plain","text/html"]` | 记录响应体的 Content-Type 白名单 |
