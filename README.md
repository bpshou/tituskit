# Titus Kit

Go 工具库，提供常用的基础设施组件，开箱即用。

## 安装

```bash
go get github.com/bpshou/tituskit
```

**要求：** Go 1.26+

## 包一览

| 包 | 说明 |
|---|------|
| [`core/auth`](#auth) | JWT 令牌生成与解析 |
| [`core/config`](#config) | 配置文件加载（Viper） |
| [`core/crontab`](#crontab) | 定时任务调度（秒级精度） |
| [`core/db`](#db) | 数据库连接（MySQL/PostgreSQL）与代码生成 |
| [`core/deploader`](#deploader) | 依赖模块拓扑加载 |
| [`core/log`](#log) | 结构化日志 + OpenTelemetry 链路追踪 |
| [`core/notice`](#notice) | 桌面通知 |
| [`core/rester`](#rester) | HTTP 客户端（Resty v3） |
| [`core/tmpl`](#tmpl) | Go 模板引擎（Sprig 函数） |
| [`gin/resp`](#ginresp) | Gin 统一响应与 OpenAI 兼容错误体 |

## 使用示例

### Auth

JWT 令牌生成与解析，签名算法固定为 HS256，解析时锁定算法并校验 issuer。

```go
import "github.com/bpshou/tituskit/core/auth"

// 创建签发/解析器
j := auth.NewJwt(auth.Jwt[any]{
    Issuer:     "my-app",
    SigningKey: []byte("my-secret-key"),
})

// 生成令牌（audience 是 []string）
token, err := j.GenerateJwtToken("user-001", []string{"app"}, 24*time.Hour)

// 解析令牌
subject, audience, err := j.ParseJwtToken(token)

// 使用自定义 Claims
type MyClaims struct {
    jwt.RegisteredClaims
    Role string `json:"role"`
}

custom := auth.NewJwt(auth.Jwt[MyClaims]{
    Issuer:     "my-app",
    SigningKey: []byte("my-secret-key"),
})

claims := &MyClaims{
    RegisteredClaims: *custom.GetRegisteredClaimsDefault(24 * time.Hour),
    Role:             "admin",
}
token, err = custom.GenerateJwtTokenWithClaims(claims)

// 解析自定义 Claims，返回值可直接断言回 *MyClaims
parsed, err := custom.ParseJwtTokenWithClaims(token, &MyClaims{})
if got, ok := parsed.(*MyClaims); ok {
    _ = got.Role
}
```

### Config

基于 Viper 的配置文件加载。

```go
import "github.com/bpshou/tituskit/core/config"

config.ViperInitEnv()

err := config.LoadViperConfig(config.ViperConfig{
    Name: "app",
    Type: "yaml",
    Path: "./configs",
    Must: true, // 加载失败则 panic
})
```

### Crontab

秒级精度的定时任务调度，支持 6 位 Cron 表达式（秒 分 时 日 月 周）。

```go
import "github.com/bpshou/tituskit/core/crontab"

c := crontab.NewCron()

// AddFunc 直接返回 error：表达式写错时立刻失败，
// 不要忽略它，否则会留下一个永远不触发的任务。
id, err := c.AddFunc("*/5 * * * * *", func() {
    fmt.Println("每5秒执行")
})
if err != nil {
    log.Fatalf("register cron job failed: %v", err)
}
c.Start()

// NextRun 回读任务的下次执行时间；id 无效、未注册或调度器未启动时 ok 为 false。
if next, ok := crontab.NextRun(c, id); ok {
    fmt.Println("next run:", next.Format(time.DateTime))
}

// 解析表达式，获取接下来 n 个执行时间
parser := crontab.NewCronParser()
times, _ := parser.NextTimeListN("0 30 * * * *", 3)  // []time.Time
dates, _ := parser.NextDateListN("0 30 * * * *", 3)  // []string（time.DateTime）

// 不传次数时固定返回 DefaultNextCount（5）个
times, _ = parser.NextTimeList("0 30 * * * *")

// 语法合法但永远推算不出时间的表达式（如 2 月 31 日）会返回
// crontab.ErrNoNextTime，可用 errors.Is 判定，不会静默返回 0001-01-01。
if _, err := parser.NextTimeListN("0 0 0 31 2 *", 3); errors.Is(err, crontab.ErrNoNextTime) {
    log.Printf("该表达式永远不会触发: %v", err)
}
```

### DB

GORM 数据库连接与代码生成。config 传 nil 时使用默认配置（含基于 slog 的
GORM 日志），连接池默认 10 空闲 / 100 最大连接 / 1 小时最大存活。

```go
import "github.com/bpshou/tituskit/core/db"

// MySQL
mysql, err := db.NewMySQL("user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local", nil)

// PostgreSQL
pg, err := db.NewPostgreSQL("host=localhost user=postgres dbname=dbname sslmode=disable", nil)

// 生成 Model 和 Query 代码：path 是输出根目录，
// 传 "." 会得到 ./model 与 ./query。
// 注意：底层 gorm.io/gen 失败时直接 panic，本函数同样以 panic 报告失败。
db.GromGen(mysql, ".")
```

### Deploader

按依赖关系并发初始化模块。

```go
import "github.com/bpshou/tituskit/core/deploader"

deploader.Load([]deploader.Module{
    {Name: "config", Fn: func() { initConfig() }},
    {Name: "db", Fn: func() { initDB() }, Deps: []string{"config"}},
    {Name: "http", Fn: func() { initHTTP() }, Deps: []string{"config", "db"}},
})
```

模块名重复、依赖了不存在的模块、依赖成环（含自依赖）都会直接 panic；
成环不会退化成永久死锁。

### Log

结构化日志，集成 OpenTelemetry 链路追踪。

```go
import "github.com/bpshou/tituskit/core/log"

// 创建 Logger：IoWriter 省略时默认写 os.Stdout
logger := log.New(log.Config{
    Level:    "info",
    Format:   log.FormatJSON, // 或 log.FormatText
    IoWriter: os.Stdout,
})
logger.Info("服务启动")

// 初始化 OpenTelemetry，返回清理函数
shutdown := log.InitTracer("my-service")
defer shutdown()
```

### Notice

发送桌面通知。

```go
import "github.com/bpshou/tituskit/core/notice"

// 带声音的通知
notice.SendNoticeAlert(ctx, "标题", "内容")

// 无声音的通知
notice.SendNoticeNotify(ctx, "标题", "内容")
```

### Rester

基于 Resty v3 的 HTTP 客户端，内置重试和超时机制。

```go
import "github.com/bpshou/tituskit/core/rester"

// 使用默认客户端（300s 超时，3 次重试），GetRequest 不接受参数
resp, err := rester.GetRequest().Get("https://api.example.com/data")

// 自定义客户端：Timeout/RetryCount 等字段必须在第一次请求之前设置，
// 配置只在首次使用时生效一次；需要多套配置就用多个 New()。
client := rester.New()
client.Timeout = 10 * time.Second
client.RetryCount = 2
resp, err = client.GetRequest().Get("https://api.example.com/data")

// SSE 流式请求（内部 SetResponseDoNotParse(true)）
resp, err = rester.GetSSERequest().Get("https://api.example.com/stream")
```

### Tmpl

基于 Sprig 函数的 Go 模板引擎，用于代码生成。

```go
import "github.com/bpshou/tituskit/core/tmpl"

engine := tmpl.NewGenEngine()

if err := engine.AddTemplate("user", "./templates/user.gotmpl"); err != nil {
    log.Fatalf("加载模板失败: %v", err)
}

// 出错返回 error，不会终止宿主进程；writer 传 nil 时写 os.Stdout
if err := engine.GenCode(os.Stdout, "user", map[string]any{
    "Name": "张三",
    "Age":  25,
}); err != nil {
    log.Fatalf("渲染模板失败: %v", err)
}
```

### gin/resp

Gin 的统一响应封装，提供两套响应体格式。

```go
import "github.com/bpshou/tituskit/gin/resp"

// Format A：Web 前端统一响应 {"code":0,"message":"ok","data":...}
resp.Success(c, "ok", gin.H{"id": 1})
resp.Error(c, http.StatusBadRequest, 40001, "参数错误")

// Format B：OpenAI 兼容错误体 {"error":{"message":"...","type":"..."}}
// 注意 AbortWithError 的参数顺序是 (状态码, errorType, message)
resp.BadRequest(c, "missing field: model")   // 400 invalid_request_error
resp.Unauthorized(c, "invalid api key")      // 401 invalid_request_error
resp.RateLimit(c)                            // 429 rate_limit_error
resp.TokenExhausted(c)                       // 429 insufficient_quota
```

## 目录结构

```
core/
├── auth/           # JWT 令牌
├── config/         # 配置加载
├── crontab/        # 定时任务
├── db/             # 数据库
├── deploader/      # 依赖加载
├── log/            # 日志与追踪
├── notice/         # 桌面通知
├── rester/         # HTTP 客户端
└── tmpl/           # 模板引擎
gin/
└── resp/           # Gin 统一响应
```

## License

MIT
