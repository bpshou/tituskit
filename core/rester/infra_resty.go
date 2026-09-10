package rester

import (
	"time"

	"log/slog"

	"resty.dev/v3"
)

type Client struct {
	RestyClient   *resty.Client
	Timeout       time.Duration
	RetryWaitTime time.Duration
	RetryCount    int
	Debug         bool
}

const (
	Timeout       = 300 * time.Second // 超时时间
	RetryWaitTime = 3 * time.Second   // 重试等待时间
	RetryCount    = 3                 // 重试次数
)

// 创建泛型实例
func New() *Client {
	return &Client{
		RestyClient:   resty.New(),
		Timeout:       Timeout,
		RetryWaitTime: RetryWaitTime,
		RetryCount:    RetryCount,
	}
}

// 获取客户端实例
func (c *Client) GetClient() *resty.Client {
	return c.RestyClient.
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
}

// 获取HTTP请求实例
func (c *Client) GetRequest() *resty.Request {
	c.RestyClient = c.GetClient()
	return c.RestyClient.R()
}

// 获取SSE请求实例
func (c *Client) GetSSERequest() *resty.Request {
	c.RestyClient = c.GetClient()
	return c.RestyClient.R().SetResponseDoNotParse(true)
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
