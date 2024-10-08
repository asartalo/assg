package commands

import (
	"html/template"
	"path"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/content"
	"github.com/asartalo/assg/internal/generator"
)

type TemplateContent struct {
	content.FrontMatter
	Content template.HTML
}

func Build(srcDir, outputDir string, includeDrafts bool, now time.Time) error {
	config, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return err
	}

	if outputDir != "" {
		config.OutputDirectory = outputDir
	}
	config.IncludeDrafts = includeDrafts
	gen, err := generator.New(config)

	if err != nil {
		return err
	}

	return gen.Build(now)
}
