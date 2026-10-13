package tmpl

import (
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type GenEngine struct {
	Tmpl *template.Template
}

func NewGenEngine() *GenEngine {
	return &GenEngine{
		Tmpl: newTemplate(),
	}
}

func newTemplate() *template.Template {
	return template.New("name").Funcs(sprig.FuncMap())
}

// template 返回可用的模板集合，零值 GenEngine 也能直接使用。
func (c *GenEngine) template() *template.Template {
	if c.Tmpl == nil {
		c.Tmpl = newTemplate()
	}
	return c.Tmpl
}

// AddTemplate 读取并解析模板文件。
//
// 出错时返回 error 而不是终止进程：库不应该替宿主决定 os.Exit。
// 解析失败时不会把半成品模板写回 c.Tmpl，但 text/template 的 New(name)
// 已经把该名字注册进模板集合，因此这个名字不会再变成可用模板。
func (c *GenEngine) AddTemplate(name string, filename string) error {
	// 读取文件流
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("tmpl: 读取模板文件 %q 失败: %w", filename, err)
	}

	// 解析模板
	tmpl, err := c.template().New(name).Parse(string(fileBytes))
	if err != nil {
		return fmt.Errorf("tmpl: 解析模板 %q 失败: %w", name, err)
	}
	c.Tmpl = tmpl

	return nil
}

// GenCode 渲染指定模板并写入 writer，writer 为 nil 时写入 os.Stdout。
func (c *GenEngine) GenCode(writer io.Writer, name string, data any) error {
	if writer == nil {
		writer = os.Stdout
	}

	if err := c.template().ExecuteTemplate(writer, name, data); err != nil {
		return fmt.Errorf("tmpl: 渲染模板 %q 失败: %w", name, err)
	}

	return nil
}
