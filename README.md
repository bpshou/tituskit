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
| [`core/log`](#log) | 结构化日志 + OpenTelemetry 链路追踪 |
| [`core/notice`](#notice) | 桌面通知 |
| [`core/rester`](#rester) | HTTP 客户端（Resty v3） |
| [`core/tmpl`](#tmpl) | Go 模板引擎（Sprig 函数） |

## 使用示例

### Auth

JWT 令牌生成与解析，支持泛型自定义 Claims。

```go
import "github.com/bpshou/tituskit/core/auth"

// 生成令牌
jwt := auth.NewJwt("my-secret-key")
token, err := jwt.GenerateJwtToken("user-001", "app", 24*time.Hour)

// 解析令牌
subject, audience, err := jwt.ParseJwtToken(token)

// 使用自定义 Claims
type MyClaims struct {
    jwt.RegisteredClaims
    Role string `json:"role"`
}
token, err = jwt.GenerateJwtTokenWithClaims(&MyClaims{
    Role: "admin",
}, "user-001", "app", 24*time.Hour)
```

### Config

基于 Viper 的配置文件加载。

```go
import "github.com/bpshou/tituskit/core/config"

config.ViperInitEnv()

err := config.LoadViperConfig(config.ViperConfig{
    Name:     "app",
    FileType: "yaml",
    Path:     "./configs",
    Must:     true, // 加载失败则 panic
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

GORM 数据库连接与代码生成。

```go
import "github.com/bpshou/tituskit/core/db"

// MySQL
mysql, err := db.NewMySQL("user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local", nil)

// PostgreSQL
pg, err := db.NewPostgreSQL("host=localhost user=postgres dbname=dbname sslmode=disable", nil)

// 生成 Model 和 Query 代码
db.GromGen(mysql, "./query")
```

### Log

结构化日志，集成 OpenTelemetry 链路追踪。

```go
import "github.com/bpshou/tituskit/core/log"

// 创建 Logger
logger := log.New(log.Config{
    Level:  log.InfoLevel,
    Format: log.JSON, // 或 log.Text
})
logger.Info("服务启动")

// 初始化 OpenTelemetry
shutdown, _ := log.InitTracer("my-service")
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

// 使用默认客户端（300s 超时，3 次重试）
resp, err := rester.GetRequest("https://api.example.com/data", nil, nil)

// 自定义客户端
client := rester.NewClient(rester.ClientConfig{
    Timeout: 10 * time.Second,
    Retry:   2,
})
resp, err = client.GetRequest("https://api.example.com/data", nil, nil)

// SSE 流式请求
resp, err = rester.GetSSERequest("https://api.example.com/stream", nil, nil)
```

### Tmpl

基于 Sprig 函数的 Go 模板引擎，用于代码生成。

```go
import "github.com/bpshou/tituskit/core/tmpl"

engine := tmpl.NewGenEngine()
engine.AddTemplate("user", "./templates/user.gotmpl")
engine.GenCode(os.Stdout, "user", map[string]interface{}{
    "Name": "张三",
    "Age":  25,
})
```

## 目录结构

```
core/
├── auth/           # JWT 令牌
├── config/         # 配置加载
├── crontab/        # 定时任务
├── db/             # 数据库
├── log/            # 日志与追踪
├── notice/         # 桌面通知
├── rester/         # HTTP 客户端
└── tmpl/           # 模板引擎
```

## License

MIT
