package db

import (
	"testing"

	"gorm.io/gorm"
)

// 非法 DSN 时构造函数应返回错误并给出 nil 的 *gorm.DB。
func TestNewMySQLInvalidDSN(t *testing.T) {
	got, err := NewMySQL("::not-a-dsn::", nil)
	if err == nil {
		t.Fatal("非法 DSN 应返回错误")
	}
	if got != nil {
		t.Fatalf("出错时应返回 nil, 实际 %v", got)
	}
}

func TestNewPostgreSQLInvalidDSN(t *testing.T) {
	got, err := NewPostgreSQL("::not-a-dsn::", nil)
	if err == nil {
		t.Fatal("非法 DSN 应返回错误")
	}
	if got != nil {
		t.Fatalf("出错时应返回 nil, 实际 %v", got)
	}
}

// 显式传入 config 时同样走通初始化路径（不覆盖调用方配置）。
func TestNewMySQLWithProvidedConfig(t *testing.T) {
	_, err := NewMySQL("::not-a-dsn::", &gorm.Config{DisableAutomaticPing: true})
	if err == nil {
		t.Fatal("非法 DSN 应返回错误")
	}
}

func TestNewPostgreSQLWithProvidedConfig(t *testing.T) {
	_, err := NewPostgreSQL("::not-a-dsn::", &gorm.Config{DisableAutomaticPing: true})
	if err == nil {
		t.Fatal("非法 DSN 应返回错误")
	}
}
