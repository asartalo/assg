package server

import (
	"fmt"
	"os"
	"path"

	"github.com/asartalo/assg/internal/config"
)

func LoadServeConfiguration(srcDir string, includeDrafts bool) (*config.Config, error) {
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
	config.BaseURL = fmt.Sprintf("http://localhost:%d", config.ServerConfig.Port)

	return config, nil
}
