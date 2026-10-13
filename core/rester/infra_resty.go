package rester

import (
	"sync"
	"time"

	"log/slog"

	"resty.dev/v3"
)

// Client 包装 resty 客户端，内置超时与重试默认值。
//
// 配置字段（Timeout/RetryWaitTime/RetryCount/Debug）必须在第一次发起请求
// 之前设置：底层客户端与回调只在首次使用时初始化一次，之后再改字段不会生效。
// 需要多套配置时请用 New() 构造多个 Client。
type Client struct {
	// RestyClient 是底层 resty 客户端，首次调用 GetClient 时创建。
	RestyClient *resty.Client

	Timeout       time.Duration
	RetryWaitTime time.Duration
	RetryCount    int
	Debug         bool

	// once 保证底层客户端与错误回调只初始化一次。
	// resty 的 OnError/OnInvalid/OnPanic 是"追加"语义，每次请求都重新配置
	// 会让回调无限累积（每条错误会重复打印 N 次，且永不释放），因此不能在
	// 每次请求时重建配置；这同时也消除了对 RestyClient 字段的并发写。
	once sync.Once
}

const (
	Timeout       = 300 * time.Second // 超时时间
	RetryWaitTime = 3 * time.Second   // 重试等待时间
	RetryCount    = 3                 // 重试次数
)

// New 创建带默认配置的 Client
func New() *Client {
	return &Client{
		RestyClient:   resty.New(),
		Timeout:       Timeout,
		RetryWaitTime: RetryWaitTime,
		RetryCount:    RetryCount,
	}
}

// GetClient 返回底层 resty 客户端，首次调用时按当前字段值完成初始化。
// 该方法可安全并发调用。
func (c *Client) GetClient() *resty.Client {
	c.once.Do(func() {
		// 零值 Client 也能直接使用；New() 构造的实例这里不会重复创建。
		if c.RestyClient == nil {
			c.RestyClient = resty.New()
		}
		// resty 的 Set* 返回同一个指针并原地生效，因此不再写回字段，
		// 避免并发调用时的字段竞争。
		c.RestyClient.
			SetTimeout(c.Timeout).
			SetRetryCount(c.RetryCount).
			SetRetryWaitTime(c.RetryWaitTime).
			SetDebug(c.Debug).
			OnError(func(req *resty.Request, err error) {
				slog.ErrorContext(req.Context(), "Send request error",
					slog.Any("error", err),
				)
			}).
			OnInvalid(func(req *resty.Request, err error) {
				slog.ErrorContext(req.Context(), "Send request invalid",
					slog.Any("error", err),
				)
			}).
			OnPanic(func(req *resty.Request, err error) {
				slog.ErrorContext(req.Context(), "Send request panic",
					slog.Any("error", err),
				)
			})
	})
	return c.RestyClient
}

// GetRequest 获取HTTP请求实例
func (c *Client) GetRequest() *resty.Request {
	return c.GetClient().R()
}

// GetSSERequest 获取SSE请求实例
func (c *Client) GetSSERequest() *resty.Request {
	return c.GetClient().R().SetResponseDoNotParse(true)
}

var defaultClient = New()

// GetRequest returns a standard HTTP request from the default client.
func GetRequest() *resty.Request {
	return defaultClient.GetRequest()
}

// GetSSERequest returns an SSE request (with SetResponseDoNotParse(true)) from the default client.
func GetSSERequest() *resty.Request {
	return defaultClient.GetSSERequest()
}
