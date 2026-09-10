package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────── Format B: AI proxy ────────────────────────────
// OpenAI-compatible error envelope: {"error": {"message": "...", "type": "..."}}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// AbortWithError sends a custom Format A error (arbitrary status/type/message).
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
	AbortWithError(c, http.StatusBadRequest, message, "invalid_request_error")
}

// Unauthorized returns 401 with type "invalid_request_error".
func Unauthorized(c *gin.Context, message string) {
	AbortWithError(c, http.StatusUnauthorized, message, "invalid_request_error")
}

// RateLimit returns 429 with type "rate_limit_error".
func RateLimit(c *gin.Context) {
	AbortWithError(c, http.StatusTooManyRequests, "rate limit exceeded, please try again later", "rate_limit_error")
}

// TokenExhausted returns 429 with type "insufficient_quota".
func TokenExhausted(c *gin.Context) {
	AbortWithError(c, http.StatusTooManyRequests, "token limit exceeded, please recharge", "insufficient_quota")
}
