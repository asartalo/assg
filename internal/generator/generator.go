package generator

import (
	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/template"
)

type Generator struct {
	Config *config.Config
	Tmpl   *template.Engine
}

func New(cfg *config.Config) *Generator {
	return &Generator{
		Config: cfg,
		Tmpl:   template.New(),
	}
}

func (g *Generator) Generate(outputDir string) error {
	return nil
}
