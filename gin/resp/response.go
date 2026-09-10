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
