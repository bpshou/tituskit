package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────── Format B: AI proxy ────────────────────────────
// OpenAI-compatible error envelope: {"error": {"message": "...", "type": "..."}}

// 错误类型常量，避免调用 AbortWithError 时把 type 和 message 传反。
const (
	ErrorTypeInvalidRequest    = "invalid_request_error"
	ErrorTypeRateLimit         = "rate_limit_error"
	ErrorTypeInsufficientQuota = "insufficient_quota"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// AbortWithError sends a Format B error with the given status, error type and
// message, then aborts the remaining handlers.
//
// 注意参数顺序：errorType 在前、message 在后。
func AbortWithError(c *gin.Context, httpStatus int, errorType string, message string) {
	c.AbortWithStatusJSON(httpStatus, ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
		},
	})
}

// BadRequest returns 400 with type "invalid_request_error".
func BadRequest(c *gin.Context, message string) {
	AbortWithError(c, http.StatusBadRequest, ErrorTypeInvalidRequest, message)
}

// Unauthorized returns 401 with type "invalid_request_error".
func Unauthorized(c *gin.Context, message string) {
	AbortWithError(c, http.StatusUnauthorized, ErrorTypeInvalidRequest, message)
}

// RateLimit returns 429 with type "rate_limit_error".
func RateLimit(c *gin.Context) {
	AbortWithError(c, http.StatusTooManyRequests, ErrorTypeRateLimit,
		"rate limit exceeded, please try again later")
}

// TokenExhausted returns 429 with type "insufficient_quota".
func TokenExhausted(c *gin.Context) {
	AbortWithError(c, http.StatusTooManyRequests, ErrorTypeInsufficientQuota,
		"token limit exceeded, please recharge")
}
