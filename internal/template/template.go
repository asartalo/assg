package template

import (
	"io"

	"github.com/CloudyKit/jet/v6"
)

type Engine struct {
	Templates *jet.Set
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) LoadTemplates(templateDir string) {
	e.Templates = jet.NewSet(
		jet.NewOSFileSystemLoader(templateDir),
		// jet.InDevelopmentMode(true),
	)
}

func (e *Engine) RenderTemplate(name string, result io.Writer, data interface{}) error {
	// return e.Templates.ExecuteTemplate(result, name, data)
	template, err := e.Templates.GetTemplate(name)
	if err != nil {
		return err
	}

	return template.Execute(result, nil, data)
}
