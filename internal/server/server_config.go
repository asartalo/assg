package server

import (
	"fmt"
	"os"
	"path"

	"codeberg.org/asartalo/assg/internal/config"
)

func LoadServeConfiguration(srcDir string, includeDrafts bool) (*config.Config, error) {
	serveDirectory, err := os.MkdirTemp("", "public-assg")
	if err != nil {
		return nil, err
	}

	conf, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return nil, err
	}

	conf.DevMode = true
	conf.OutputDirectory = serveDirectory
	conf.IncludeDrafts = includeDrafts
	conf.BaseURL = fmt.Sprintf("http://localhost:%d", conf.ServerConfig.Port)

	return conf, nil
}
