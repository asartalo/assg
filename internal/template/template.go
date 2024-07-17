package template

import (
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/Masterminds/sprig/v3"
)

type Engine struct {
	Templates *template.Template
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) LoadTemplates(templateDir string) error {
	err := filepath.WalkDir(templateDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".html" {
			relPath, err := filepath.Rel(templateDir, path)
			if err != nil {
				return err
			}

			tmpTemplate := template.New(relPath)

			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			tmpTemplate.Parse(string(contents))
			e.Templates = tmpTemplate
		}
		return nil
	})

	e.Templates = e.Templates.Funcs(sprig.FuncMap())

	return err
}

func (e *Engine) RenderTemplate(name string, result io.Writer, data interface{}) error {
	return e.Templates.ExecuteTemplate(result, name, data)
}

func (e *Engine) TemplateExists(name string) bool {
	template := e.Templates.Lookup(name)
	return template != nil
}
