package router

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"backend_go/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// requestIDSeq is an in-process monotonic sequence for request-id suffix.
// requestIDSeq 是程序內遞增序號，用於 request-id 尾碼。
var requestIDSeq uint64

// requestLogger logs one line for every HTTP request.
// requestLogger 會為每一筆 HTTP request 記錄一行日誌。
func requestLogger(log *zap.Logger, cfg model.AppConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		startAt := time.Now()
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID(startAt)
		}

		// Return request id in response header for client-side tracing.
		// 在回應 header 回傳 request id，方便客戶端追蹤。
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		var requestBodyBuf bytes.Buffer
		if cfg.HTTPLogRequestBody && c.Request != nil && c.Request.Body != nil {
			// Tee body stream so handler can consume original bytes unchanged.
			// 透過 Tee 複製 body 串流，handler 仍可讀取原始內容不受影響。
			originalBody := c.Request.Body
			c.Request.Body = &teeReadCloser{
				Reader: io.TeeReader(originalBody, &requestBodyBuf),
				Closer: originalBody,
			}
		}

		c.Next()

		latency := time.Since(startAt)
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("route", c.FullPath()),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int64("request_bytes", c.Request.ContentLength),
			zap.Int("response_bytes", c.Writer.Size()),
			zap.Duration("latency", latency),
		}
		if cfg.HTTPLogRequestBody {
			bodyRaw := requestBodyBuf.Bytes()
			bodyLogged := bodyRaw
			bodyTruncated := false
			if len(bodyRaw) > cfg.HTTPLogRequestBodyBytes {
				bodyLogged = bodyRaw[:cfg.HTTPLogRequestBodyBytes]
				bodyTruncated = true
			}
			fields = append(fields,
				zap.Int("request_body_bytes", len(bodyRaw)),
				zap.Bool("request_body_truncated", bodyTruncated),
				zap.String("request_body", string(bodyLogged)),
			)
		}

		// Include first handler error if any middleware/handler pushed it.
		// 若 middleware/handler 有錯誤，附上第一筆錯誤訊息。
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors[0].Error()))
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			log.Error("http_request", fields...)
		case status >= 400:
			log.Warn("http_request", fields...)
		default:
			log.Info("http_request", fields...)
		}
	}
}

// newRequestID builds a compact request identifier.
// newRequestID 產生簡潔的 request 識別碼。
func newRequestID(now time.Time) string {
	seq := atomic.AddUint64(&requestIDSeq, 1)
	return fmt.Sprintf("req-%d-%d", now.UTC().UnixNano(), seq)
}

// teeReadCloser keeps original Close while replacing Reader with TeeReader.
// teeReadCloser 保留原本 Close 行為，同時以 TeeReader 取代 Reader。
type teeReadCloser struct {
	io.Reader
	io.Closer
}
