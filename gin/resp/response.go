package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────── Format A: web frontend ────────────────────────────
// Envelope: {"code": int, "message": "...", "data": any}

// Response is the standard web frontend response envelope.
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Success returns HTTP 200 with code=0 and the given data.
func Success(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error returns an arbitrary HTTP status with a custom code and message.
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// Result returns an arbitrary HTTP status with a custom code and message.
func Result(c *gin.Context, httpStatus int, res Response) {
	c.JSON(httpStatus, res)
}

// ──────────────────────────── Format B: AI proxy ────────────────────────────
// OpenAI-compatible error envelope: {"error": {"message": "...", "type": "..."}}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// BadRequest returns 400 with type "invalid_request_error".
func BadRequest(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    "invalid_request_error",
		},
	})
}

// Unauthorized returns 401 with type "invalid_request_error".
func Unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    "invalid_request_error",
		},
	})
}

// RateLimit returns 429 with type "rate_limit_error".
func RateLimit(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
		Error: ErrorDetail{
			Message: "rate limit exceeded, please try again later",
			Type:    "rate_limit_error",
		},
	})
}

// TokenExhausted returns 429 with type "insufficient_quota".
func TokenExhausted(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
		Error: ErrorDetail{
			Message: "token limit exceeded, please recharge",
			Type:    "insufficient_quota",
		},
	})
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
