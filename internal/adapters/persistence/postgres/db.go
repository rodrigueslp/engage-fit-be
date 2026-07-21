package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

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
		slog.ErrorContext(ctx, "database_query_failed", "latency_ms", latencyMS)
	case l.slowThreshold > 0 && time.Since(startedAt) > l.slowThreshold && l.level >= logger.Warn:
		slog.WarnContext(ctx, "database_query_slow", "latency_ms", latencyMS)
	}
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
