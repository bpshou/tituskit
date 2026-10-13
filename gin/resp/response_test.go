package resp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func decodeErrorResponse(t *testing.T, w *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var got ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v (body=%s)", err, w.Body.String())
	}
	return got
}

// OpenAI 兼容错误体里 message 和 type 必须各就各位：
// 早期版本把 AbortWithError 的两个字符串参数传反，客户端拿到的是
// {"error":{"message":"invalid_request_error","type":"<真实错误信息>"}}。
func TestOpenAIErrorEnvelopes(t *testing.T) {
	tests := []struct {
		name       string
		write      func(c *gin.Context)
		wantStatus int
		wantType   string
		wantMsg    string
	}{
		{
			name:       "BadRequest",
			write:      func(c *gin.Context) { BadRequest(c, "missing field: model") },
			wantStatus: http.StatusBadRequest,
			wantType:   ErrorTypeInvalidRequest,
			wantMsg:    "missing field: model",
		},
		{
			name:       "Unauthorized",
			write:      func(c *gin.Context) { Unauthorized(c, "invalid api key") },
			wantStatus: http.StatusUnauthorized,
			wantType:   ErrorTypeInvalidRequest,
			wantMsg:    "invalid api key",
		},
		{
			name:       "RateLimit",
			write:      func(c *gin.Context) { RateLimit(c) },
			wantStatus: http.StatusTooManyRequests,
			wantType:   ErrorTypeRateLimit,
			wantMsg:    "rate limit exceeded, please try again later",
		},
		{
			name:       "TokenExhausted",
			write:      func(c *gin.Context) { TokenExhausted(c) },
			wantStatus: http.StatusTooManyRequests,
			wantType:   ErrorTypeInsufficientQuota,
			wantMsg:    "token limit exceeded, please recharge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newTestContext()
			tt.write(c)

			if w.Code != tt.wantStatus {
				t.Errorf("HTTP 状态码 = %d, 期望 %d", w.Code, tt.wantStatus)
			}
			got := decodeErrorResponse(t, w)
			if got.Error.Type != tt.wantType {
				t.Errorf("error.type = %q, 期望 %q", got.Error.Type, tt.wantType)
			}
			if got.Error.Message != tt.wantMsg {
				t.Errorf("error.message = %q, 期望 %q", got.Error.Message, tt.wantMsg)
			}
			if !c.IsAborted() {
				t.Error("AbortWithStatusJSON 之后上下文应处于已中断状态")
			}
		})
	}
}

func TestAbortWithErrorCustomType(t *testing.T) {
	c, w := newTestContext()
	AbortWithError(c, http.StatusForbidden, "custom_type", "custom message")

	if w.Code != http.StatusForbidden {
		t.Errorf("HTTP 状态码 = %d, 期望 %d", w.Code, http.StatusForbidden)
	}
	got := decodeErrorResponse(t, w)
	if got.Error.Type != "custom_type" || got.Error.Message != "custom message" {
		t.Errorf("错误体 = %+v", got.Error)
	}
}

func TestSuccessEnvelope(t *testing.T) {
	c, w := newTestContext()
	Success(c, "ok", gin.H{"id": 1})

	if w.Code != http.StatusOK {
		t.Errorf("HTTP 状态码 = %d, 期望 200", w.Code)
	}
	var got Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Code != 0 || got.Message != "ok" {
		t.Errorf("响应体 = %+v", got)
	}
}

func TestErrorEnvelope(t *testing.T) {
	c, w := newTestContext()
	Error(c, http.StatusBadRequest, 40001, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("HTTP 状态码 = %d, 期望 400", w.Code)
	}
	var got Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Code != 40001 || got.Message != "参数错误" || got.Data != nil {
		t.Errorf("响应体 = %+v", got)
	}
}

func TestResultEnvelope(t *testing.T) {
	c, w := newTestContext()
	Result(c, http.StatusCreated, Response{Code: 0, Message: "created", Data: "x"})

	if w.Code != http.StatusCreated {
		t.Errorf("HTTP 状态码 = %d, 期望 201", w.Code)
	}
	var got Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Message != "created" || got.Data != "x" {
		t.Errorf("响应体 = %+v", got)
	}
}
