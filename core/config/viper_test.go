package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// writeConfig 在 dir 下写入 name.type 配置文件。
func writeConfig(t *testing.T, dir, name, fileType, content string) {
	t.Helper()
	path := filepath.Join(dir, name+"."+fileType)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// resetViper 清理全局 viper 实例，避免用例之间互相污染。
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

func TestLoadViperConfig(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		content  string
	}{
		{"yaml", "yaml", "name: titus\nport: 8080\n"},
		{"json", "json", `{"name":"titus","port":8080}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			dir := t.TempDir()
			writeConfig(t, dir, "app", tt.fileType, tt.content)

			err := LoadViperConfig(ViperConfig{
				Name: "app",
				Type: tt.fileType,
				Path: dir,
			})
			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}

			if got := viper.GetString("name"); got != "titus" {
				t.Errorf("name = %q, 期望 titus", got)
			}
			if got := viper.GetInt("port"); got != 8080 {
				t.Errorf("port = %d, 期望 8080", got)
			}
		})
	}
}

func TestLoadViperConfigFromMultiplePaths(t *testing.T) {
	resetViper(t)
	empty := t.TempDir()
	dir := t.TempDir()
	writeConfig(t, dir, "app", "yaml", "name: titus\n")

	// 第一个路径为空目录，应继续在第二个路径查找。
	viper.AddConfigPath(empty)

	if err := LoadViperConfig(ViperConfig{Name: "app", Type: "yaml", Path: dir}); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if got := viper.GetString("name"); got != "titus" {
		t.Errorf("name = %q, 期望 titus", got)
	}
}

func TestLoadViperConfigMissingFile(t *testing.T) {
	resetViper(t)

	err := LoadViperConfig(ViperConfig{
		Name: "not-exist",
		Type: "yaml",
		Path: t.TempDir(),
	})
	if err == nil {
		t.Fatal("配置文件不存在时应返回错误")
	}
}

// Must 为 true 时加载失败直接 panic，而不是返回 error。
func TestLoadViperConfigMustPanics(t *testing.T) {
	resetViper(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Must=true 且加载失败时应 panic")
		}
	}()

	_ = LoadViperConfig(ViperConfig{
		Name: "not-exist",
		Type: "yaml",
		Path: t.TempDir(),
		Must: true,
	})
}

func TestLoadViperConfigInvalidContent(t *testing.T) {
	resetViper(t)
	dir := t.TempDir()
	writeConfig(t, dir, "app", "yaml", "name: [unclosed\n")

	if err := LoadViperConfig(ViperConfig{Name: "app", Type: "yaml", Path: dir}); err == nil {
		t.Fatal("配置文件内容非法时应返回错误")
	}
}

func TestViperInitEnv(t *testing.T) {
	resetViper(t)
	t.Setenv("TITUSKIT_TEST_ENV_KEY", "from-env")

	ViperInitEnv()

	if got := viper.GetString("TITUSKIT_TEST_ENV_KEY"); got != "from-env" {
		t.Errorf("环境变量未生效: %q, 期望 from-env", got)
	}
}
