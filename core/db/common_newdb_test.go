package db

import (
	"errors"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// offlineDialector 返回一个不需要真实数据库即可完成 gorm.Open 的 Dialector：
// 跳过版本探测，并配合 DisableAutomaticPing 避免建连。
func offlineDialector() gorm.Dialector {
	return mysql.New(mysql.Config{
		DSN:                       "user:pass@tcp(127.0.0.1:1)/db",
		SkipInitializeWithVersion: true,
	})
}

// 覆盖"打开成功"的完整路径：gorm.Open -> db.DB() -> 连接池默认值 -> 返回。
func TestNewDBSuccess(t *testing.T) {
	gdb, err := newDB(offlineDialector(), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("newDB 失败: %v", err)
	}
	if gdb == nil {
		t.Fatal("newDB 返回 nil")
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("获取 *sql.DB 失败: %v", err)
	}
	if got := sqlDB.Stats().MaxOpenConnections; got != 100 {
		t.Errorf("MaxOpenConnections = %d, 期望 100", got)
	}
}

// config.Logger 为 nil 时应注入基于 slog 的 Logger。
func TestNewDBInjectsSlogLogger(t *testing.T) {
	gdb, err := newDB(offlineDialector(), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("newDB 失败: %v", err)
	}

	if gdb.Config.Logger == nil {
		t.Fatal("config.Logger 为 nil 时未注入 Logger")
	}
	if _, ok := gdb.Config.Logger.(*slogGormLogger); !ok {
		t.Errorf("Logger 类型 = %T, 期望 *slogGormLogger", gdb.Config.Logger)
	}
}

// 调用方显式传入 Logger 时应原样保留。
func TestNewDBKeepsProvidedLogger(t *testing.T) {
	custom := gormlogger.Default

	gdb, err := newDB(offlineDialector(), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               custom,
	})
	if err != nil {
		t.Fatalf("newDB 失败: %v", err)
	}
	if gdb.Config.Logger != custom {
		t.Error("调用方传入的 Logger 被覆盖了")
	}
}

// 非法 Dialector 应返回错误而不是 panic。
func TestNewDBInvalidDialector(t *testing.T) {
	gdb, err := newDB(postgres.Open("::not-a-dsn::"), nil)
	if err == nil {
		t.Fatal("非法 DSN 应返回错误")
	}
	if gdb != nil {
		t.Fatalf("出错时应返回 nil, 实际 %v", gdb)
	}
}

// nilPoolDialector 是一个不设置连接池的 Dialector：
// gorm.Open 之后 db.ConnPool 仍为 nil，db.DB() 返回 gorm.ErrInvalidDB，
// 用来覆盖 newDB 把 db.DB() 错误上抛的分支。
type nilPoolDialector struct{}

func (nilPoolDialector) Name() string                    { return "nilpool" }
func (nilPoolDialector) Initialize(*gorm.DB) error       { return nil }
func (nilPoolDialector) Migrator(*gorm.DB) gorm.Migrator { return nil }

func (nilPoolDialector) DataTypeOf(*schema.Field) string { return "" }

func (nilPoolDialector) DefaultValueOf(*schema.Field) clause.Expression { return nil }

func (nilPoolDialector) BindVarTo(clause.Writer, *gorm.Statement, interface{}) {}

func (nilPoolDialector) QuoteTo(clause.Writer, string) {}

func (nilPoolDialector) Explain(string, ...interface{}) string { return "" }

// 拿不到 *sql.DB 时，newDB 应把错误返回给调用方而不是 panic。
func TestNewDBInvalidConnPool(t *testing.T) {
	gdb, err := newDB(nilPoolDialector{}, &gorm.Config{DisableAutomaticPing: true})
	if !errors.Is(err, gorm.ErrInvalidDB) {
		t.Fatalf("期望 gorm.ErrInvalidDB, 实际 %v", err)
	}
	if gdb != nil {
		t.Fatalf("出错时应返回 nil, 实际 %v", gdb)
	}
}
