package middleware

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func HTTPMetrics() gin.HandlerFunc {
	meter := otel.Meter("engagefit/http")
	requests, _ := meter.Int64Counter("engagefit.http.server.requests", metric.WithDescription("Total de requisicoes HTTP"))
	duration, _ := meter.Float64Histogram(
		"engagefit.http.server.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duracao das requisicoes HTTP"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	inFlight, _ := meter.Int64UpDownCounter("engagefit.http.server.active_requests", metric.WithDescription("Requisicoes HTTP em andamento"))

	return func(c *gin.Context) {
		startedAt := time.Now()
		method := c.Request.Method
		inFlight.Add(c.Request.Context(), 1, metric.WithAttributes(attribute.String("http.request.method", method)))
		defer inFlight.Add(c.Request.Context(), -1, metric.WithAttributes(attribute.String("http.request.method", method)))

		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		attrs := metric.WithAttributes(
			attribute.String("http.request.method", method),
			attribute.String("http.route", route),
			attribute.String("http.response.status_code", strconv.Itoa(c.Writer.Status())),
		)
		requests.Add(c.Request.Context(), 1, attrs)
		duration.Record(c.Request.Context(), time.Since(startedAt).Seconds(), attrs)
	}
}

func MetricsAccess(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expectedToken == "" {
			c.Next()
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if len(provided) != len(expectedToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
