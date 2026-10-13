package tmpl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAddTemplateAndGenCode(t *testing.T) {
	dir := t.TempDir()
	path := writeTemplate(t, dir, "user.gotmpl", "Hello {{ .Name | upper }}!")

	engine := NewGenEngine()
	if err := engine.AddTemplate("user", path); err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}

	var buf bytes.Buffer
	if err := engine.GenCode(&buf, "user", map[string]any{"Name": "titus"}); err != nil {
		t.Fatalf("渲染模板失败: %v", err)
	}
	if got := buf.String(); got != "Hello TITUS!" {
		t.Errorf("渲染结果 = %q, 期望 %q", got, "Hello TITUS!")
	}
}

// 库不应该在出错时调用 log.Fatal 终止宿主进程，而应返回 error。
func TestAddTemplateMissingFileReturnsError(t *testing.T) {
	engine := NewGenEngine()
	err := engine.AddTemplate("missing", filepath.Join(t.TempDir(), "not-exist.gotmpl"))
	if err == nil {
		t.Fatal("文件不存在时应返回错误")
	}
	if !strings.Contains(err.Error(), "读取模板文件") {
		t.Errorf("错误信息缺少上下文: %v", err)
	}
}

func TestAddTemplateParseErrorReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := writeTemplate(t, dir, "bad.gotmpl", "{{ .Name ")

	engine := NewGenEngine()
	err := engine.AddTemplate("bad", path)
	if err == nil {
		t.Fatal("模板语法错误时应返回错误")
	}
	if !strings.Contains(err.Error(), "解析模板") {
		t.Errorf("错误信息缺少上下文: %v", err)
	}
}

// 回归：解析失败时早期实现会把 c.Tmpl 置为 nil，导致引擎彻底不可用。
func TestFailedParseKeepsEngineUsable(t *testing.T) {
	dir := t.TempDir()
	goodPath := writeTemplate(t, dir, "good.gotmpl", "ok: {{ .Value }}")
	badPath := writeTemplate(t, dir, "bad.gotmpl", "{{ .Value ")

	engine := NewGenEngine()
	if err := engine.AddTemplate("good", goodPath); err != nil {
		t.Fatalf("添加模板失败: %v", err)
	}
	if err := engine.AddTemplate("bad", badPath); err == nil {
		t.Fatal("非法模板应返回错误")
	}

	var buf bytes.Buffer
	if err := engine.GenCode(&buf, "good", map[string]any{"Value": 1}); err != nil {
		t.Fatalf("解析失败后已有模板应仍可渲染: %v", err)
	}
	if got := buf.String(); got != "ok: 1" {
		t.Errorf("渲染结果 = %q", got)
	}
}

func TestGenCodeUnknownTemplate(t *testing.T) {
	engine := NewGenEngine()
	var buf bytes.Buffer
	if err := engine.GenCode(&buf, "not-registered", nil); err == nil {
		t.Fatal("未注册的模板名应返回错误")
	}
}

// 零值 GenEngine 不应 panic。
func TestZeroValueEngine(t *testing.T) {
	var engine GenEngine
	var buf bytes.Buffer
	if err := engine.GenCode(&buf, "anything", nil); err == nil {
		t.Fatal("空引擎渲染应返回错误而不是 panic")
	}
	if engine.Tmpl == nil {
		t.Fatal("GenCode 后应初始化模板集合")
	}
}

// writer 为 nil 时回退到 os.Stdout，不应 panic。
func TestGenCodeNilWriter(t *testing.T) {
	dir := t.TempDir()
	path := writeTemplate(t, dir, "empty.gotmpl", "ok")

	engine := NewGenEngine()
	if err := engine.AddTemplate("empty", path); err != nil {
		t.Fatal(err)
	}
	if err := engine.GenCode(nil, "empty", nil); err != nil {
		t.Fatalf("nil writer 应回退到 stdout: %v", err)
	}
}

func TestSprigFunctionsAvailable(t *testing.T) {
	dir := t.TempDir()
	path := writeTemplate(t, dir, "sprig.gotmpl", `{{ list "a" "b" | join "," }}`)

	engine := NewGenEngine()
	if err := engine.AddTemplate("sprig", path); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := engine.GenCode(&buf, "sprig", nil); err != nil {
		t.Fatalf("Sprig 函数不可用: %v", err)
	}
	if got := buf.String(); got != "a,b" {
		t.Errorf("渲染结果 = %q, 期望 a,b", got)
	}
}

func TestAddTemplateMultipleTemplates(t *testing.T) {
	dir := t.TempDir()
	a := writeTemplate(t, dir, "a.gotmpl", "A")
	b := writeTemplate(t, dir, "b.gotmpl", "B")

	engine := NewGenEngine()
	if err := engine.AddTemplate("a", a); err != nil {
		t.Fatal(err)
	}
	if err := engine.AddTemplate("b", b); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{"a": "A", "b": "B"} {
		var buf bytes.Buffer
		if err := engine.GenCode(&buf, name, nil); err != nil {
			t.Fatalf("渲染 %s 失败: %v", name, err)
		}
		if buf.String() != want {
			t.Errorf("模板 %s 渲染结果 = %q, 期望 %q", name, buf.String(), want)
		}
	}
}
