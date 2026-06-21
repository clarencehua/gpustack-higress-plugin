# GPUStack Higress 插件开发指南

本文档介绍如何为 GPUStack Higress 插件系统开发新的 Proxy-Wasm 插件。

## 目录

- [概述](#概述)
- [开发环境准备](#开发环境准备)
- [插件架构](#插件架构)
- [创建新插件](#创建新插件)
- [核心概念](#核心概念)
- [开发流程](#开发流程)
- [测试](#测试)
- [构建和部署](#构建和部署)
- [最佳实践](#最佳实践)
- [示例插件](#示例插件)

## 概述

GPUStack Higress 插件系统基于 Proxy-Wasm 标准，使用 Go 语言开发，编译为 WebAssembly (.wasm) 格式。插件可以拦截、修改和观察 HTTP 请求和响应，为 AI API 网关提供流量处理、可观测性和增强功能。

### 核心特性

- **请求/响应拦截**：可以读取和修改 HTTP 请求和响应
- **流式处理**：支持流式请求体和响应体的实时处理
- **配置管理**：通过 YAML/JSON 配置控制插件行为
- **状态管理**：可以在请求处理过程中存储和传递上下文信息
- **元数据输出**：可以将处理结果写入 Envoy Filter State

## 开发环境准备

### 必需工具

1. **Go 1.24.4+**
   ```bash
   # 下载 Go 1.24.4
   wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
   sudo rm -rf /usr/local/go
   sudo tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
   ```

2. **Python 3.10+**
   ```bash
   python3 --version  # 确认版本
   ```

3. **oras** (可选，用于获取远程插件)
   ```bash
   brew install oras  # macOS
   # 或参考官方文档安装
   ```

### 项目设置

```bash
# 克隆项目
git clone <repository-url>
cd gpustack-higress-plugin

# 创建虚拟环境
make venv

# 安装开发依赖
make dev

# 验证环境
go version
python3 --version
```

## 插件架构

### 项目结构

```
gpustack-higress-plugin/
├── extensions/                    # Go 插件源码目录
│   ├── your-plugin-name/          # 你的插件目录
│   │   ├── main.go                # 主程序文件
│   │   ├── main_test.go           # 测试文件
│   │   ├── go.mod                 # Go 模块配置
│   │   └── VERSION                # 版本文件
│   ├── Makefile                   # 构建脚本
│   └── remote_plugins.yaml        # 远程插件配置
├── gpustack_higress_plugins/      # Python 包
├── scripts/                       # 构建和工具脚本
├── go.work                        # Go workspace 配置
└── Makefile                       # 主构建文件
```

### 插件生命周期

1. **初始化阶段**：`init()` 函数注册插件回调
2. **配置解析**：`parseConfig()` 解析 YAML/JSON 配置
3. **请求处理**：
   - `onHttpRequestHeaders()` - 处理请求头
   - `onStreamingRequestBody()` - 处理流式请求体
4. **响应处理**：
   - `onHttpResponseHeaders()` - 处理响应头
   - `onStreamingResponseBody()` - 处理流式响应体

## 创建新插件

### 步骤 1：创建插件目录

```bash
mkdir -p extensions/your-plugin-name
cd extensions/your-plugin-name
```

### 步骤 2：创建 VERSION 文件

```bash
echo "1.0.0" > VERSION
```

### 步骤 3：初始化 Go 模块

```bash
go mod init github.com/gpustack/gpustack-higress-plugins/extensions/your-plugin-name
```

### 步骤 4：创建 main.go

```go
package main

import (
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
    "github.com/higress-group/wasm-go/pkg/wrapper"
    "github.com/tidwall/gjson"
)

// 插件配置
type PluginConfig struct {
    Enable bool `json:"enable"`
}

// 插件名称
const pluginName = "your-plugin-name"

func main() {}

func init() {
    wrapper.SetCtx(
        pluginName,
        wrapper.ParseConfig(parseConfig),
        wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
        wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
    )
}

// 解析配置
func parseConfig(json gjson.Result, config *PluginConfig) error {
    config.Enable = json.Get("enable").Bool()
    return nil
}

// 处理请求头
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
    if !config.Enable {
        return types.ActionContinue
    }

    // 获取所有请求头
    headers, err := proxywasm.GetHttpRequestHeaders()
    if err != nil {
        return types.ActionContinue
    }

    // 处理请求头...
    for _, header := range headers {
        // header[0] 是名称，header[1] 是值
        proxywasm.LogInfof("Header: %s = %s", header[0], header[1])
    }

    return types.ActionContinue
}

// 处理响应头
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
    if !config.Enable {
        return types.ActionContinue
    }

    // 获取所有响应头
    headers, err := proxywasm.GetHttpResponseHeaders()
    if err != nil {
        return types.ActionContinue
    }

    // 处理响应头...
    for _, header := range headers {
        proxywasm.LogInfof("Response Header: %s = %s", header[0], header[1])
    }

    return types.ActionContinue
}
```

### 步骤 5：创建 go.mod

```go
module github.com/gpustack/gpustack-higress-plugins/extensions/your-plugin-name

go 1.24.4

require (
    github.com/higress-group/proxy-wasm-go-sdk v0.0.0-20251103120604-77e9cce339d2
    github.com/higress-group/wasm-go v1.0.7-0.20251209122854-7e766df5675c
    github.com/stretchr/testify v1.9.0
    github.com/tidwall/gjson v1.18.0
    github.com/tidwall/sjson v1.2.5
)
```

### 步骤 6：更新 go.work

在项目根目录的 `go.work` 文件中添加你的插件：

```go
go 1.24.4

use (
    ./extensions/gpustack-token-usage
    ./extensions/your-plugin-name  // 添加这行
)
```

## 核心概念

### 1. 配置解析

配置通过 `gjson.Result` 传递，使用 gjson 语法访问：

```go
func parseConfig(json gjson.Result, config *PluginConfig) error {
    // 简单字段
    config.Enable = json.Get("enable").Bool()
    config.Name = json.Get("name").String()

    // 嵌套对象
    config.Timeout = int(json.Get("settings.timeout").Int())

    // 数组
    if tags := json.Get("tags").Array(); len(tags) > 0 {
        for _, tag := range tags {
            config.Tags = append(config.Tags, tag.String())
        }
    }

    return nil
}
```

### 2. 请求头处理

```go
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
    headers, err := proxywasm.GetHttpRequestHeaders()
    if err != nil {
        proxywasm.LogErrorf("Failed to get request headers: %v", err)
        return types.ActionContinue
    }

    // 查找特定头
    for _, header := range headers {
        if header[0] == "content-type" {
            proxywasm.LogInfof("Content-Type: %s", header[1])
        }
    }

    // 添加新头
    proxywasm.AddHttpRequestHeader("X-Custom-Header", "value")

    return types.ActionContinue
}
```

### 3. 请求体处理

```go
func onStreamingRequestBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool) []byte {
    // 获取已缓冲的数据
    buffer, _ := ctx.GetContext("body_buffer").([]byte)

    // 追加当前块
    buffer = append(buffer, chunk...)

    // 存储更新后的缓冲区
    ctx.SetContext("body_buffer", buffer)

    // 如果是流结束，处理完整请求体
    if isEndStream {
        // 处理 buffer...
        proxywasm.LogInfof("Complete request body: %s", string(buffer))
        ctx.SetContext("body_buffer", []byte{})
    }

    // 返回原始块（不修改请求）
    return chunk
}
```

### 4. Filter State（过滤器状态）

可以将数据写入 Envoy Filter State，供其他过滤器使用：

```go
import "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

func setProperty(key string, value string) {
    if err := proxywasm.SetProperty([]string{key}, []byte(value)); err != nil {
        proxywasm.LogErrorf("Failed to set property %s: %v", key, err)
    }
}

// 使用示例
setProperty("my_plugin_data", "some_value")
```

### 5. 上下文管理

在请求处理过程中存储和检索上下文信息：

```go
// 存储上下文
ctx.SetContext("key", value)

// 获取上下文
value, exists := ctx.GetContext("key")
if exists {
    // 使用 value
}
```

## 开发流程

### 1. 设计插件功能

- 明确插件要做什么
- 确定需要拦截的请求/响应阶段
- 设计配置结构
- 规划输出数据

### 2. 实现核心逻辑

- 实现配置解析
- 实现请求/响应处理函数
- 添加错误处理
- 实现日志记录

### 3. 编写测试

```go
package main

import (
    "encoding/json"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/tidwall/gjson"
)

func TestParseConfig(t *testing.T) {
    // 创建测试配置
    configData := map[string]interface{}{
        "enable": true,
        "name":   "test",
    }
    data, _ := json.Marshal(configData)
    jsonResult := gjson.ParseBytes(data)

    var config PluginConfig
    err := parseConfig(jsonResult, &config)

    require.NoError(t, err)
    require.True(t, config.Enable)
    require.Equal(t, "test", config.Name)
}
```

### 4. 本地测试

```bash
# 运行单个插件测试
make -C extensions test PLUGIN_NAME=your-plugin-name

# 运行所有测试
make test
```

### 5. 构建插件

```bash
# 构建单个插件
make -C extensions build PLUGIN_NAME=your-plugin-name

# 构建所有插件
make -C extensions build-all
```

## 测试

### 单元测试

测试配置解析、工具函数等：

```go
func TestParseConfig(t *testing.T) {
    makeConfig := func(v map[string]interface{}) gjson.Result {
        data, _ := json.Marshal(v)
        return gjson.ParseBytes(data)
    }

    t.Run("default values", func(t *testing.T) {
        raw := makeConfig(map[string]interface{}{
            "enable": true,
        })
        var cfg PluginConfig
        err := parseConfig(raw, &cfg)
        require.NoError(t, err)
        require.True(t, cfg.Enable)
    })
}
```

### 集成测试

使用 Higress 测试框架测试完整的请求-响应流程：

```go
import "github.com/higress-group/wasm-go/pkg/test"

func TestCompleteFlow(t *testing.T) {
    test.RunTest(t, func(t *testing.T) {
        // 创建测试主机
        config := map[string]interface{}{
            "enable": true,
        }
        data, _ := json.Marshal(config)
        host, status := test.NewTestHost(data)
        defer host.Reset()

        // 处理请求头
        action := host.CallOnHttpRequestHeaders([][2]string{
            {":method", "GET"},
            {":path", "/test"},
        })
        require.Equal(t, types.ActionContinue, action)

        // 完成请求
        host.CompleteHttp()
    })
}
```

### 运行测试

```bash
# 运行单个插件测试
make -C extensions test PLUGIN_NAME=your-plugin-name

# 运行所有插件测试
make -C extensions test-all

# 使用 -v 查看详细输出
cd extensions/your-plugin-name
go test -v ./...
```

## 构建和部署

### 本地构建

```bash
# 构建插件
make -C extensions build PLUGIN_NAME=your-plugin-name

# 构建 Python 包
make build
```

### Docker 构建

```bash
# 构建 Docker 镜像
make image

# 运行容器
docker run -p 8080:8080 gpustack/higress-plugins:latest
```

### 部署到 Higress

1. **创建 WasmPlugin 资源**：

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: your-plugin-name
  namespace: higress-system
spec:
  url: http://plugin-server:8080/wasm-plugins/your-plugin-name/1.0.0/plugin.wasm
  defaultConfig:
    enable: true
```

2. **启动插件服务器**：

```bash
# 使用 CLI
gpustack-plugins start --port 8080

# 或使用 Docker
docker run -d -p 8080:8080 gpustack/higress-plugins:latest
```

## 最佳实践

### 1. 错误处理

```go
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
    headers, err := proxywasm.GetHttpRequestHeaders()
    if err != nil {
        proxywasm.LogErrorf("Failed to get request headers: %v", err)
        return types.ActionContinue  // 不要中断请求处理
    }

    // 处理逻辑...
    return types.ActionContinue
}
```

### 2. 性能优化

```go
// 避免在热路径中分配内存
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}

func processChunk(chunk []byte) []byte {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)

    // 使用 buf 处理数据...
    return result
}
```

### 3. 日志记录

```go
import "github.com/higress-group/wasm-go/pkg/log"

// 使用不同级别的日志
log.Debugf("Debug message: %s", msg)
log.Infof("Info message: %s", msg)
log.Warnf("Warning message: %s", msg)
log.Errorf("Error message: %s", msg)
```

### 4. 配置验证

```go
func parseConfig(json gjson.Result, config *PluginConfig) error {
    // 验证必需字段
    if !json.Get("name").Exists() {
        return fmt.Errorf("missing required field: name")
    }

    // 设置默认值
    if config.Timeout <= 0 {
        config.Timeout = 30  // 默认 30 秒
    }

    return nil
}
```

### 5. 版本管理

- 使用语义化版本号（MAJOR.MINOR.PATCH）
- 在 VERSION 文件中维护版本号
- 遵循项目的版本管理策略

## 示例插件

### 简单日志插件

```go
package main

import (
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
    "github.com/higress-group/wasm-go/pkg/wrapper"
    "github.com/tidwall/gjson"
)

const pluginName = "simple-logger"

type PluginConfig struct {
    LogHeaders bool `json:"logHeaders"`
    LogBody    bool `json:"logBody"`
}

func main() {}

func init() {
    wrapper.SetCtx(
        pluginName,
        wrapper.ParseConfig(parseConfig),
        wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
    )
}

func parseConfig(json gjson.Result, config *PluginConfig) error {
    config.LogHeaders = json.Get("logHeaders").Bool()
    config.LogBody = json.Get("logBody").Bool()
    return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
    if config.LogHeaders {
        headers, _ := proxywasm.GetHttpRequestHeaders()
        for _, h := range headers {
            proxywasm.LogInfof("%s: %s", h[0], h[1])
        }
    }
    return types.ActionContinue
}
```

### 请求头注入插件

```go
package main

import (
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
    "github.com/higress-group/wasm-go/pkg/wrapper"
    "github.com/tidwall/gjson"
)

const pluginName = "header-injector"

type PluginConfig struct {
    Headers map[string]string `json:"headers"`
}

func main() {}

func init() {
    wrapper.SetCtx(
        pluginName,
        wrapper.ParseConfig(parseConfig),
        wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
    )
}

func parseConfig(json gjson.Result, config *PluginConfig) error {
    config.Headers = make(map[string]string)
    headers := json.Get("headers")
    headers.ForEach(func(key, value gjson.Result) bool {
        config.Headers[key.String()] = value.String()
        return true
    })
    return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
    for name, value := range config.Headers {
        proxywasm.AddHttpRequestHeader(name, value)
    }
    return types.ActionContinue
}
```

## 常见问题

### Q: 如何调试插件？

A: 使用 `proxywasm.Log*` 系列函数输出日志，查看 Envoy 的日志：

```go
proxywasm.LogInfof("Debug info: %v", someValue)
proxywasm.LogErrorf("Error: %v", err)
```

### Q: 如何处理二进制数据？

A: 请求体和响应体是字节数组，可以直接操作：

```go
func onStreamingRequestBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool) []byte {
    // chunk 是字节数组，可以包含二进制数据
    // 处理数据...
    return chunk
}
```

### Q: 如何与其他插件通信？

A: 使用 Filter State 共享数据：

```go
// 插件 A 写入数据
proxywasm.SetProperty([]string{"shared_key"}, []byte("value"))

// 插件 B 读取数据
value, err := proxywasm.GetProperty([]string{"shared_key"})
```

### Q: 如何处理大请求体？

A: 使用流式处理，避免一次性加载整个请求体：

```go
func onStreamingRequestBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool) []byte {
    // 逐块处理，避免内存溢出
    if len(chunk) > config.MaxChunkSize {
        chunk = chunk[:config.MaxChunkSize]
    }
    // 处理块...
    return chunk
}
```

## 参考资源

- [Proxy-Wasm 规范](https://github.com/proxy-wasm/spec)
- [Higress 文档](https://higress.io/)
- [proxy-wasm-go-sdk](https://github.com/higress-group/proxy-wasm-go-sdk)
- [wasm-go](https://github.com/higress-group/wasm-go)

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 编写代码和测试
4. 运行测试确保通过
5. 提交 Pull Request

确保遵循项目的代码规范和最佳实践。
