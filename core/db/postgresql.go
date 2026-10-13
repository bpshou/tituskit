package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgreSQL 创建 PostgreSQL 连接。
//
// config 为 nil 时使用默认配置；config.Logger 为 nil 时注入基于 slog 的日志。
func NewPostgreSQL(dsn string, config *gorm.Config) (*gorm.DB, error) {
	return newDB(postgres.Open(dsn), config)
}
