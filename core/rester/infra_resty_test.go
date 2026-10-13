package rester

import (
	"context"

	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"resty.dev/v3"
)

// recordingHandler 收集日志消息，用于断言 resty 回调是否被正确注册。
type recordingHandler struct {
	mu       sync.Mutex
	messages []string
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, r.Message)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func (h *recordingHandler) count(sub string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, m := range h.messages {
		if strings.Contains(m, sub) {
			n++
		}
	}
	return n
}

func captureSlog(t *testing.T) *recordingHandler {
	t.Helper()
	h := &recordingHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return h
}

// newTestClient 返回一个关闭重试的客户端，避免重试等待拖慢用例。
func newTestClient() *Client {
	c := New()
	c.RetryCount = 0
	c.Timeout = 2 * time.Second
	return c
}

// 127.0.0.1:1 必然连接失败，用于触发 OnError 回调。
const unreachableURL = "http://127.0.0.1:1/"

func TestGetRequestDoesNotAccumulateErrorHooks(t *testing.T) {
	handler := captureSlog(t)
	client := newTestClient()

	fail := func() {
		t.Helper()
		resp, err := client.GetRequest().Get(unreachableURL)
		if err == nil {
			t.Fatal("期望请求失败，实际成功了")
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	// resty 的 OnError/OnInvalid/OnPanic 是"追加"语义：
	// 若每次请求都重新配置客户端，回调会不断累积，
	// 同一条错误会被重复打印且永不释放。
	fail()
	perRequest := handler.count("Send request error")
	if perRequest == 0 {
		t.Fatal("请求失败时 OnError 回调没有触发")
	}

	const requests = 3
	for i := 1; i < requests; i++ {
		fail()
		total := handler.count("Send request error")
		if want := perRequest * (i + 1); total != want {
			t.Fatalf("第 %d 次请求后错误日志共 %d 条，期望 %d 条（错误回调在累积）",
				i+1, total, want)
		}
	}
}

// OnInvalid 回调：请求在发送前就判定为非法（例如 URL 无法解析）。
func TestInvalidHookIsRegistered(t *testing.T) {
	handler := captureSlog(t)
	client := newTestClient()

	resp, err := client.GetRequest().Get("://bad-url")
	if err == nil {
		t.Fatal("非法 URL 应返回错误")
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if got := handler.count("Send request invalid"); got != 1 {
		t.Fatalf("OnInvalid 回调触发 %d 次，期望 1 次", got)
	}
}

// OnPanic 回调：请求执行过程中 panic 时记录日志，并重新抛出 panic。
func TestPanicHookIsRegistered(t *testing.T) {
	handler := captureSlog(t)
	client := newTestClient()

	client.GetClient().SetRequestMiddlewares(func(*resty.Client, *resty.Request) error {
		panic("boom")
	})

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		resp, err := client.GetRequest().Get(unreachableURL)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		_ = err
	}()

	if recovered == nil {
		t.Fatal("resty 会重新抛出 panic，调用方应能 recover")
	}
	if got := handler.count("Send request panic"); got != 1 {
		t.Fatalf("OnPanic 回调触发 %d 次，期望 1 次", got)
	}
}

func TestSuccessfulRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"titus"}`))
	}))
	defer srv.Close()

	client := newTestClient()

	var out struct {
		Name string `json:"name"`
	}
	resp, err := client.GetRequest().SetResult(&out).Get(srv.URL)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("状态码 = %d, 期望 200", resp.StatusCode())
	}
	if out.Name != "titus" {
		t.Errorf("响应未反序列化到 SetResult: %+v", out)
	}
}

// SSE 请求设置 SetResponseDoNotParse(true)，原始响应体必须保持可读。
func TestSSERequestKeepsRawBody(t *testing.T) {
	const payload = "data: hello\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	client := newTestClient()

	resp, err := client.GetSSERequest().Get(srv.URL)
	if err != nil {
		t.Fatalf("SSE 请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取 SSE 响应体失败: %v", err)
	}
	if string(body) != payload {
		t.Errorf("响应体 = %q, 期望 %q", string(body), payload)
	}
}

func TestDefaultClientConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if GetRequest() == nil {
				t.Error("GetRequest 返回 nil")
			}
			if GetSSERequest() == nil {
				t.Error("GetSSERequest 返回 nil")
			}
		}()
	}
	wg.Wait()
}

func TestGetClientReturnsSameInstance(t *testing.T) {
	client := New()
	first := client.GetClient()
	if first == nil {
		t.Fatal("GetClient 返回 nil")
	}
	if second := client.GetClient(); first != second {
		t.Fatal("重复调用 GetClient 应返回同一个底层客户端")
	}
}

func TestZeroValueClientIsUsable(t *testing.T) {
	var client Client
	if client.GetRequest() == nil {
		t.Fatal("零值 Client 应可直接使用")
	}
}

// New() 构造的实例应立即持有底层客户端，保持既有用法可用。
func TestNewCreatesRestyClientEagerly(t *testing.T) {
	if client := New(); client.RestyClient == nil {
		t.Fatal("New() 应已创建底层 resty 客户端")
	}
}

// 默认配置应来自包级常量。
func TestNewUsesDefaultConfig(t *testing.T) {
	client := New()
	if client.Timeout != Timeout {
		t.Errorf("Timeout = %v, 期望 %v", client.Timeout, Timeout)
	}
	if client.RetryWaitTime != RetryWaitTime {
		t.Errorf("RetryWaitTime = %v, 期望 %v", client.RetryWaitTime, RetryWaitTime)
	}
	if client.RetryCount != RetryCount {
		t.Errorf("RetryCount = %d, 期望 %d", client.RetryCount, RetryCount)
	}
}

// 首次使用之后再改字段不会生效：配置只应用一次。
func TestConfigIsAppliedOnce(t *testing.T) {
	client := New()
	client.Timeout = 5 * time.Second

	if got := client.GetClient().Timeout(); got != 5*time.Second {
		t.Fatalf("首次初始化未应用配置: Timeout = %v", got)
	}

	client.Timeout = 30 * time.Second
	if got := client.GetClient().Timeout(); got != 5*time.Second {
		t.Fatalf("配置应只生效一次: Timeout = %v, 期望 %v", got, 5*time.Second)
	}
}

// GetRequest / GetSSERequest 每次返回彼此独立的请求对象。
func TestRequestsAreIndependent(t *testing.T) {
	client := newTestClient()

	first := client.GetRequest()
	second := client.GetRequest()
	if first == second {
		t.Fatal("每次调用应返回新的请求对象")
	}
	if first == client.GetSSERequest() {
		t.Fatal("SSE 请求应是独立对象")
	}
}

// 请求超时后应返回错误（Timeout 配置真实生效）。
func TestRequestTimeout(t *testing.T) {
	const serverDelay = 500 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(serverDelay)
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	client := New()
	client.RetryCount = 0
	client.Timeout = 50 * time.Millisecond

	start := time.Now()
	resp, err := client.GetRequest().Get(srv.URL)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("超时应返回错误")
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	// 在服务端返回之前就中断，说明是超时而不是等到响应。
	if elapsed >= serverDelay {
		t.Fatalf("超时未生效，耗时 %v（服务端延迟 %v）", elapsed, serverDelay)
	}
}
