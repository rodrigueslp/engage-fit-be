package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
)

type windowEntry struct {
	count     int
	expiresAt time.Time
}

type WindowRateLimiter struct {
	mu      sync.Mutex
	entries map[string]windowEntry
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewWindowRateLimiter(limit int, window time.Duration) *WindowRateLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &WindowRateLimiter{entries: map[string]windowEntry{}, limit: limit, window: window, now: time.Now}
}

func (l *WindowRateLimiter) Allow(keys ...string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for key, entry := range l.entries {
		if !entry.expiresAt.After(now) {
			delete(l.entries, key)
		}
	}

	var retryAfter time.Duration
	for _, key := range keys {
		entry := l.entries[key]
		if entry.count >= l.limit && entry.expiresAt.After(now) {
			remaining := entry.expiresAt.Sub(now)
			if remaining > retryAfter {
				retryAfter = remaining
			}
		}
	}
	if retryAfter > 0 {
		return false, retryAfter
	}

	for _, key := range keys {
		entry := l.entries[key]
		if !entry.expiresAt.After(now) {
			entry = windowEntry{expiresAt: now.Add(l.window)}
		}
		entry.count++
		l.entries[key] = entry
	}
	return true, 0
}

func JSONRateLimit(limiter *WindowRateLimiter, identityField string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if strings.Contains(err.Error(), "request body too large") || errors.As(err, &maxBytesError) {
				apiresponse.AbortError(c, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
				return
			}
			apiresponse.AbortError(c, http.StatusBadRequest, "invalid_request", "invalid request")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		keys := []string{"ip:" + c.ClientIP()}
		var payload map[string]any
		if json.Unmarshal(body, &payload) == nil {
			if identity, ok := payload[identityField].(string); ok && strings.TrimSpace(identity) != "" {
				sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(identity))))
				keys = append(keys, "identity:"+hex.EncodeToString(sum[:]))
			}
		}

		allowed, retryAfter := limiter.Allow(keys...)
		if !allowed {
			seconds := int(retryAfter.Round(time.Second).Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			apiresponse.AbortError(c, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests")
			return
		}
		c.Next()
	}
}

func BodySizeLimit(defaultBytes, importBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := defaultBytes
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == "/api/v1/imports" {
			limit = importBytes
		}
		if limit > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
