package db

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 懒连接：不建立真实连接，只拿到 *gorm.DB。
// 用于验证 GromGen 在无法读取表结构时的失败方式。
func lazyDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(
		mysql.New(mysql.Config{
			DSN:                       "user:pass@tcp(127.0.0.1:1)/db",
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	if err != nil {
		t.Fatalf("构造懒连接失败: %v", err)
	}
	return gdb
}

// gorm.io/gen 的 Execute 没有 error 返回值，失败时直接 panic；
// GromGen 继承这一行为，调用方必须自行 recover。
func TestGromGenPanicsWithoutDatabase(t *testing.T) {
	gdb := lazyDB(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("无法读取表结构时 GromGen 应 panic")
		}
	}()

	GromGen(gdb, t.TempDir())
}
