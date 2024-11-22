package commands

import (
	"path"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/generator"
)

func Build(srcDir, outputDir string, includeDrafts bool, verbose bool, now time.Time) error {
	config, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return err
	}

	if outputDir != "" {
		config.OutputDirectory = outputDir
	}
	config.IncludeDrafts = includeDrafts
	gen, err := generator.New(config, verbose)

	if err != nil {
		return err
	}

	return gen.Build(now)
}
