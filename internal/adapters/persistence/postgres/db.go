package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
)

type PoolConfig struct {
	MaxOpenConnections int
	MaxIdleConnections int
	MaxLifetime        time.Duration
	MaxIdleTime        time.Duration
}

func Open(databaseURL string) (*gorm.DB, error) {
	gormLogger := safeGORMLogger{level: logger.Warn, slowThreshold: 500 * time.Millisecond}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{Logger: gormLogger, DisableAutomaticPing: true})
	if err != nil {
		return nil, err
	}
	if err := db.Use(tracing.NewPlugin(tracing.WithoutQueryVariables())); err != nil {
		return nil, err
	}
	return db, nil
}

type safeGORMLogger struct {
	level         logger.LogLevel
	slowThreshold time.Duration
}

func (l safeGORMLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.level = level
	return l
}

func (l safeGORMLogger) Info(ctx context.Context, _ string, _ ...interface{}) {
	if l.level >= logger.Info {
		slog.DebugContext(ctx, "database_info")
	}
}

func (l safeGORMLogger) Warn(ctx context.Context, _ string, _ ...interface{}) {
	if l.level >= logger.Warn {
		slog.WarnContext(ctx, "database_warning")
	}
}

func (l safeGORMLogger) Error(ctx context.Context, _ string, _ ...interface{}) {
	if l.level >= logger.Error {
		slog.ErrorContext(ctx, "database_error")
	}
}

func (l safeGORMLogger) Trace(ctx context.Context, startedAt time.Time, _ func() (string, int64), err error) {
	latencyMS := time.Since(startedAt).Milliseconds()
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= logger.Error:
		errorKind, sqlState := safeDatabaseErrorDetails(err)
		slog.ErrorContext(ctx, "database_query_failed", "latency_ms", latencyMS, "error_kind", errorKind, "sqlstate", sqlState)
	case l.slowThreshold > 0 && time.Since(startedAt) > l.slowThreshold && l.level >= logger.Warn:
		slog.WarnContext(ctx, "database_query_slow", "latency_ms", latencyMS)
	}
}

func safeDatabaseErrorDetails(err error) (string, string) {
	if errors.Is(err, context.Canceled) {
		return "request_canceled", ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded", ""
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch {
		case strings.HasPrefix(postgresError.Code, "08"):
			return "connection", postgresError.Code
		case postgresError.Code == "23503":
			return "foreign_key_violation", postgresError.Code
		case postgresError.Code == "23505":
			return "unique_violation", postgresError.Code
		case postgresError.Code == "40001":
			return "serialization_failure", postgresError.Code
		case postgresError.Code == "40P01":
			return "deadlock", postgresError.Code
		case postgresError.Code == "57014":
			return "query_canceled", postgresError.Code
		default:
			return "postgres_error", postgresError.Code
		}
	}
	lowerError := strings.ToLower(err.Error())
	if strings.Contains(lowerError, "65535") && strings.Contains(lowerError, "parameter") {
		return "parameter_limit", ""
	}
	return "database_error", ""
}

func ConfigurePool(db *gorm.DB, config PoolConfig) (*sql.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(config.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.MaxIdleTime)
	return sqlDB, nil
}
