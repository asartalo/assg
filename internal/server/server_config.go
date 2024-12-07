package server

import (
	"fmt"
	"os"
	"path"

	"github.com/asartalo/assg/internal/config"
)

func LoadServeConfiguration(port, srcDir string, includeDrafts bool) (*config.Config, error) {
	serveDirectory, err := os.MkdirTemp("", "public-assg")
	if err != nil {
		return nil, err
	}

	config, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return nil, err
	}

	config.DevMode = true
	config.OutputDirectory = serveDirectory
	config.IncludeDrafts = includeDrafts
	config.BaseURL = fmt.Sprintf("http://localhost:%s", port)

	return config, nil
}
