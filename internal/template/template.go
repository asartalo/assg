package template

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
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

	return err
}

func (e *Engine) RenderTemplate(name string, result io.Writer, data interface{}) error {
	return e.Templates.ExecuteTemplate(result, name, data)

}

// func templateFuncs() template.FuncMap {
// 	return template.FuncMap{
// 		// Define custom template functions
// 	}
// }
