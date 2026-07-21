package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

const RequestIDKey = "request_id"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !validRequestID.MatchString(requestID) {
			requestID = generateRequestID()
		}

		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		requestID, _ := c.Get(RequestIDKey)
		attrs := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
		}
		spanContext := trace.SpanContextFromContext(c.Request.Context())
		if spanContext.IsValid() {
			attrs = append(attrs, "trace_id", spanContext.TraceID().String(), "span_id", spanContext.SpanID().String())
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error_count", len(c.Errors))
		}

		switch status := c.Writer.Status(); {
		case status >= 500:
			slog.Error("http_request_finished", attrs...)
		case status >= 400:
			slog.Warn("http_request_finished", attrs...)
		default:
			slog.Info("http_request_finished", attrs...)
		}
	}
}

func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
