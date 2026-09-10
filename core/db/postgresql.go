package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultGormSlowThreshold = 200 * time.Millisecond

// MySQL连接
func NewPostgreSQL(dsn string, config *gorm.Config) (*gorm.DB, error) {
	if config == nil {
		config = &gorm.Config{}
	}
	if config.Logger == nil {
		config.Logger = newSlogGormLogger(defaultGormSlowThreshold)
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxIdleConns 设置空闲连接池中连接的最大数量。
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了可以重新使用连接的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

type slogGormLogger struct {
	slowThreshold time.Duration
}

func newSlogGormLogger(slowThreshold time.Duration) gormlogger.Interface {
	return &slogGormLogger{slowThreshold: slowThreshold}
}

func (l *slogGormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *slogGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	slog.InfoContext(ctx, fmt.Sprintf(msg, data...))
}

func (l *slogGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	slog.WarnContext(ctx, fmt.Sprintf(msg, data...))
}

func (l *slogGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	slog.ErrorContext(ctx, fmt.Sprintf(msg, data...))
}

func (l *slogGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {

	sql, rows := fc()
	elapsed := time.Since(begin)
	attrs := []any{
		slog.String("event", "gorm"),
		slog.String("sql", sql),
		slog.Int64("rows", rows),
		slog.Duration("elapsed", elapsed),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.WarnContext(ctx, "gorm trace", attrs...)
			return
		}
		slog.ErrorContext(ctx, "gorm trace", attrs...)
		return
	}

	if elapsed > l.slowThreshold {
		slog.WarnContext(ctx, "gorm slow sql", attrs...)
		return
	}

	slog.InfoContext(ctx, "gorm sql", attrs...)
}