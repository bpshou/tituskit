package tmpl

import (
	"io"
	"log"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type GenEngine struct {
	Tmpl *template.Template
}

func NewGenEngine() *GenEngine {
	return &GenEngine{
		Tmpl: template.New("name").Funcs(sprig.FuncMap()),
	}
}

// 添加模板
func (c *GenEngine) AddTemplate(name string, filename string) *GenEngine {
	// 读取文件流
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	// 解析模板
	c.Tmpl, err = c.Tmpl.New(name).Parse(string(fileBytes))
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return c
}

// 生成代码
func (c *GenEngine) GenCode(writer io.Writer, name string, data any) error {
	if writer == nil {
		writer = os.Stdout
	}

	if err := c.Tmpl.ExecuteTemplate(writer, name, data); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}
