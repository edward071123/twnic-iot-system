package response

import "github.com/gin-gonic/gin"

// Package response standardizes JSON response envelopes.
// response 套件統一 JSON 回應格式。

// JSON writes standard JSON response payload with status code.
// JSON 依指定狀態碼輸出標準 JSON 回應。
func JSON(c *gin.Context, status int, payload gin.H) {
	c.JSON(status, payload)
}

// Error writes unified API error payload shape.
// Error 以統一錯誤格式輸出 API 回應。
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   message,
	})
}
