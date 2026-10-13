package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMySQL 创建 MySQL 连接。
//
// config 为 nil 时使用默认配置；config.Logger 为 nil 时注入基于 slog 的日志。
func NewMySQL(dsn string, config *gorm.Config) (*gorm.DB, error) {
	return newDB(mysql.Open(dsn), config)
}
