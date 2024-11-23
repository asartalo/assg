package commands

import (
	"fmt"
	"log"
	"net/http"

	"os"
	"path"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/generator"
)

func Serve(srcDir string, includeDrafts bool) error {
	port := "8080"
	now := time.Now()
	serveDirectory, err := os.MkdirTemp("", "public-assg")
	if err != nil {
		return err
	}

	config, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return err
	}

	config.OutputDirectory = serveDirectory
	config.IncludeDrafts = includeDrafts
	config.BaseURL = fmt.Sprintf("http://localhost:%s", port)

	gen, err := generator.New(config, false)
	if err != nil {
		return err
	}

	err = gen.Build(now)
	if err != nil {
		return err
	}

	http.Handle("/", http.FileServer(http.Dir(serveDirectory)))
	log.Printf("Serving %s on HTTP port: %s\n", serveDirectory, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

	return nil
}
