# Dipperick 的轻量化 Gin 风格 Web 框架（带 AI 助手示例）

本项目是一个从零实现的、面向学习与实践的轻量化 Web 框架，借鉴 Gin 的核心设计理念，包含：

- HTTP 基础与 `net/http` 生态对接（`Engine` 实现 `http.Handler`）
- 路由前缀树（Trie）与参数匹配（`:name`、`*filepath`）
- 路由分组（`RouterGroup`）与中间件链（`Next()` 机制）
- 上下文 `Context` 封装（请求/响应、渲染、控制流）
- 模板渲染（`html/template`）与静态资源服务
- 日志中间件与错误恢复中间件（panic 安全）
- CORS 中间件（开发期跨域便捷）
- 一个带前端页面的 AI 助手 Demo（含聊天与清空记录功能）

适合用于理解 Go Web 框架的核心构建要点，也可以作为个人小项目的起点。

---

## 目录结构

```
go_web_like_gin/
  go.mod                 # example 模块
  main.go                # 演示与入口：路由、模板、AI Demo
  README.md
  gee/                   # 框架本体（独立 go module）
    go.mod
    context.go           # 上下文 Context 封装
    cors.go              # CORS 中间件
    gee.go               # Engine / RouterGroup 实现
    logger.go            # 日志中间件
    recovery.go          # 错误恢复中间件（panic -> 500）
    router.go            # 路由器，Trie 接入、匹配、分发
    trie.go              # 路由 Trie 结构与算法
    router_test.go       # 路由相关单元测试
    gee_test.go          # 分组相关单元测试
  templates/
    guide.tmpl           # 指南页（中文）
    ai.tmpl              # AI 助手聊天页（中文）
  static/
    style.css            # 简洁深色主题样式
```

---

## 运行与快速体验

```zsh
cd go_web_like_gin
# 构建
go build ./...
# 运行 Demo
go run main.go
```

访问：

- 指南页（中文）：http://localhost:9999/
- ![指南页截图](../img_1.png)
- AI 聊天页：http://localhost:9999/ai
- ![聊天页截图](../img_2.png)
- 分组 API 示例：
  - `GET /api/v1/ping`
  - `GET /api/v1/users/:name`
- panic 演示（配合 Recovery 中间件）：`GET /panic`

命令行调用示例：

```zsh
# 聊天：表单字段 q 为问题内容
curl -X POST http://localhost:9999/api/ai/chat -d 'q=Hello Dipperick'

# 清空聊天记录
curl -X POST http://localhost:9999/api/ai/reset
```

> 注：AI Demo 为教学演示，返回“反转后的输入内容”，并将“你 / 助手”的对话保存在内存切片中，使用 RWMutex 保证并发安全。

---

## HTTP 基础与 Engine 设计

- Go 标准库 `net/http` 使用 `Handler` 接口驱动服务，核心是：
  - `type Handler interface { ServeHTTP(ResponseWriter, *Request) }`
- 本框架的 `Engine` 实现了 `ServeHTTP`：
  - 在 `ServeHTTP` 中：
    - 根据请求路径收集中间件（按路由分组前缀匹配）
    - 构造 `Context`，注入中间件链、Engine 指针
    - 调用路由器进行匹配与处理

这样，你可以直接用：

```go
log.Fatal(r.Run(":9999"))
```

启动一个 HTTP 服务。

---

## Context 设计与能力

文件：`gee/context.go`

- 将 `http.ResponseWriter` 与 `*http.Request` 封装为 `Context`：
  - 便捷读取：`Param()`、`Query()`、`PostForm()`
  - 便捷输出：`String()`、`JSON()`、`Data()`、`HTML()`
  - 控制响应：`Status()`、`SetHeader()`
  - 中间件链控制：`Next()`、`Fail()`（中断后续并返回错误）
- 模板渲染：`HTML(status, name, data)` 会使用 `engine.htmlTemplates` 渲染模板。

---

## 路由前缀树（Trie）

文件：`gee/trie.go`、`gee/router.go`

- 路由模式解析：`/p/:name`、`/assets/*filepath`。
- `parsePattern`：
  - 将路径按 `/` 切分，跳过空段
  - `*` 只能出现一次且为最后一段（出现时提前停止）
- Trie 节点：`node{ pattern, part, children, isWild }`
  - `isWild` 表示 `:` 或 `*` 通配
  - `insert`：逐层创建/下沉
  - `search`：逐层匹配，遇 `*` 提前匹配成功
- 匹配得到的 `node.pattern` 再与请求路径做反向填参，得到 `Params`。

复杂度：匹配时间复杂度与路径段数（深度）成正比，能在大规模路由下保持稳定表现。

---

## 路由器与 404 处理

文件：`gee/router.go`

- `addRoute(method, pattern, handler)`：
  - 写入 Trie
  - 以键 `method-pattern` 存入 `handlers` 映射
- `handle(c *Context)`：
  - 匹配到节点则设置 `Params` 并将最终处理函数追加到 `c.handlers`
  - 未匹配则追加一个 404 处理器：`404 NOT FOUND: <path>`

---

## 路由分组与中间件链

文件：`gee/gee.go`

- `RouterGroup`：包含 `prefix`、`middlewares`、`parent`、`engine`
  - `Group("/api")`、`Group("/v1")` 可嵌套：最终前缀叠加
  - `Use(mw...)` 给某个组挂中间件
- `Engine.ServeHTTP`：
  - 遍历所有分组，凡是 `req.URL.Path` 以该组前缀开头，则收集其 `middlewares`
  - 构造 `Context` 后进入路由处理
- 中间件执行：
  - `Context.Next()` 实现责任链：每个中间件/处理器均需显式调用 `Next()` 才会继续

中间件示例：

- `Logger()`：记录响应状态码与耗时
- `Recovery()`：捕获 panic、打印堆栈并返回 500，服务不中断
- `CORS()`：开发期默认放开跨域（生产环境可收紧）

---

## 模板与静态资源

- 模板：
  - `SetFuncMap(template.FuncMap)` 注入模板函数（例：`now`）
  - `LoadHTMLGlob("templates/*")` 绑定模板
  - `Context.HTML()` 渲染模板
- 静态资源：
  - `RouterGroup.Static("/static", "static")`
  - 使用 `http.FileServer` 与 `StripPrefix` 提供文件

本项目内置：

- `templates/guide.tmpl`：中文指南页，列出路由与使用方式
- `templates/ai.tmpl`：AI 聊天页面（AJAX 调用后端）
- `static/style.css`：深色主题样式

---

## 错误恢复与日志

- 错误恢复（`recovery.go`）
  - `defer` + `recover()` 捕获业务处理中的 panic
  - 打印调用栈（`runtime.Callers` + `FuncForPC`）
  - 统一返回 `{ "message": "Internal Server Error" }` 或通过 `Fail()` 返回 JSON
- 日志（`logger.go`）
  - 记录 `StatusCode`、`RequestURI`、处理耗时

---

## CORS 支持

- 中间件 `CORS()` 默认设置：
  - `Access-Control-Allow-Origin: *`
  - 允许常见方法、头部，`OPTIONS` 直接返回 204
- 说明：生产环境请按需限制域名、方法、头部以及 Max-Age

---

## AI 助手 Demo（前后端一体）

- 页面：`GET /ai` 使用模板渲染聊天 UI
- API：
  - `POST /api/ai/chat`，Form 字段 `q`，返回：`{ "answer": "..." }`
  - `POST /api/ai/reset`，清空对话记录，返回：`{ "ok": true }`
- 服务端逻辑：
  - 将 `q` 反转作为“助手回答”（演示用）
  - 历史对话内存存储（`messages []string`），由 `RWMutex` 保护并发
- 前端：
  - 使用 `fetch` 向后端提交 `application/x-www-form-urlencoded`
  - 聊天历史即时插入页面，可一键清空

> 想要接入真实 LLM（OpenAI/Azure/本地模型等），可以在 `POST /api/ai/chat` 的处理器中调用对应 SDK/HTTP API，并将响应结果返回。

---

## 使用指南（主流程）

1) 创建引擎与中间件：

```go
r := gee.Default()      // 内置 Logger + Recovery
r.Use(gee.CORS())       // 开发期跨域方便前端调试
```

2) 加载模板与静态资源：

```go
r.SetFuncMap(template.FuncMap{ "now": func() string { return time.Now().Format(time.RFC822) } })
r.LoadHTMLGlob("templates/*")
r.Static("/static", "static")
```

3) 路由与分组：

```go
r.GET("/", guideHandler)
api := r.Group("/api")
v1 := api.Group("/v1")
v1.GET("/ping", ping)
v1.GET("/users/:name", hello)
```

4) 处理器中使用 Context：

```go
func ping(c *gee.Context) {
  c.JSON(http.StatusOK, gee.H{"message": "pong"})
}
```

5) 启动服务：

```go
log.Fatal(r.Run(":9999"))
```

---

## 单元测试

本仓库包含路由与分组的单元测试（位于 `gee` 模块下）。建议在 `gee/` 目录下运行：

```zsh
cd gee
go test ./...
```

测试覆盖：
- `parsePattern` 的解析行为（`/p/:name`、`/assets/*filepath` 等）
- `getRoute` 的匹配行为与参数提取
- 多路由枚举与分组前缀嵌套

---

## 设计取舍与扩展建议

- 中间件顺序：按分组前缀匹配顺序收集；具体执行顺序由加入 `handlers` 的先后决定。
- 路由覆盖：`addRoute` 使用 `method-pattern` 作为键，重复注册会覆盖此前处理器；可在生产框架中增加冲突检测。
- 并发安全：
  - 框架核心为无状态（路由表只读），Demo 的聊天历史通过锁保护；
  - 更复杂的状态建议使用持久化（Redis/SQLite）与会话隔离。
- 性能：
  - Trie 匹配在大量路由下仍保持较好性能；
  - 可按需增加缓存（如静态路由缓存）、路由编译优化。
- 模板与前端：
  - 可增加布局（layout）与组件化模板，支持多主题；
  - 开发期可启用模板热加载（当前示例未实现）。
- 工程化：
  - 可增加 `Makefile`、CI、Lint、benchmarks；
  - 增加 `pprof` 与 tracing，有助于性能分析与排错。

---

## 与 Gin 的关系

- 设计借鉴 Gin，但实现简洁、可读，便于学习掌握 Web 框架底层实现；
- 若用于生产，建议直接使用成熟框架（Gin、Echo、Fiber 等），或在此基础上补齐更多工程化能力。

---

## 致谢与版权

- 关键思想参考了GO开源社区常见的路由 Trie 与中间件链实现方式；
- 代码主要用于学习与分享，欢迎在此基础上扩展功能并注明来源。
