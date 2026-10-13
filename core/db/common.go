package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultGormSlowThreshold = 200 * time.Millisecond

// newGormConfig 补齐 nil 配置并注入 slog 日志，MySQL / PostgreSQL 共用。
func newGormConfig(config *gorm.Config) *gorm.Config {
	if config == nil {
		config = &gorm.Config{}
	}
	if config.Logger == nil {
		config.Logger = newSlogGormLogger(defaultGormSlowThreshold)
	}
	return config
}

// newDB 打开连接、设置连接池并返回 *gorm.DB，MySQL / PostgreSQL 共用。
//
// 抽成独立函数是为了让"打开成功"的分支可以被单元测试覆盖：
// 传入一个不需要真实连接的 Dialector（例如 SkipInitializeWithVersion +
// DisableAutomaticPing）即可走完整条路径。
func newDB(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, newGormConfig(config))
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	applyPoolDefaults(sqlDB)

	return db, nil
}

// applyPoolDefaults 设置连接池默认参数，MySQL / PostgreSQL 共用。
func applyPoolDefaults(sqlDB *sql.DB) {
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量。
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了可以重新使用连接的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)
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
