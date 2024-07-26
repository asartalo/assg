package commands

import (
	"html/template"
	"path"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/generator"
)

type TemplateContent struct {
	generator.FrontMatter
	Content template.HTML
}

func Build(srcDir, outputDir string, includeDrafts bool) error {
	config, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return err
	}

	if outputDir != "" {
		config.OutputDirectory = outputDir
	}
	config.IncludeDrafts = includeDrafts
	gen := generator.New(config)

	return gen.Build()
}
