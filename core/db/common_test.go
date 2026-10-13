package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 不产生真实连接的假 driver，仅用于拿到 *sql.DB 验证连接池配置。
type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("stub driver: not used")
}

type stubConnector struct{}

func (stubConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errors.New("stub connector: not used")
}

func (stubConnector) Driver() driver.Driver { return stubDriver{} }

// config 为 nil 时应补齐默认配置并注入 slog 日志，而不是让调用方拿到 nil Logger。
func TestNewGormConfigNil(t *testing.T) {
	cfg := newGormConfig(nil)
	if cfg == nil {
		t.Fatal("newGormConfig(nil) 返回 nil")
	}
	if cfg.Logger == nil {
		t.Fatal("Logger 未注入，MySQL/PostgreSQL 日志行为会与预期不一致")
	}
}

// 调用方显式传入 Logger 时不应被覆盖。
func TestNewGormConfigKeepsProvidedLogger(t *testing.T) {
	custom := gormlogger.Default
	cfg := newGormConfig(&gorm.Config{Logger: custom})
	if cfg.Logger != custom {
		t.Fatal("调用方传入的 Logger 被覆盖了")
	}
}

// PostgreSQL 版本原本会补齐 nil 配置，MySQL 版本没有，这里验证两者共用同一套默认值。
func TestApplyPoolDefaults(t *testing.T) {
	sqlDB := sql.OpenDB(stubConnector{})
	defer sqlDB.Close()

	applyPoolDefaults(sqlDB)

	if got := sqlDB.Stats().MaxOpenConnections; got != 100 {
		t.Errorf("MaxOpenConnections = %d, 期望 100", got)
	}
}
