package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/generator"
	"github.com/jaschaephraim/lrserver"
)

func buildForServer(config *config.Config, now time.Time) error {
	log.Println("Building site...")
	gen, err := generator.New(config, false)
	if err != nil {
		return err
	}

	err = gen.Build(now)
	if err != nil {
		return err
	}

	return nil
}

type Server struct {
	Config *config.Config
	SrcDir string
	done   chan bool
}

func NewServer(srcDir string, includeDrafts bool) (*Server, error) {
	config, err := LoadServeConfiguration("8080", srcDir, includeDrafts)
	if err != nil {
		return nil, err
	}

	return &Server{
		Config: config,
		SrcDir: srcDir,
		done:   make(chan bool),
	}, nil
}

func (s *Server) Start() error {
	port := "8080"
	srcDir := s.SrcDir
	includeDrafts := s.Config.IncludeDrafts

	config, err := LoadServeConfiguration(port, srcDir, includeDrafts)
	if err != nil {
		return err
	}

	serveDirectory := config.OutputDirectory

	lr := lrserver.New(lrserver.DefaultName, lrserver.DefaultPort)
	go lr.ListenAndServe()

	var serverStarted bool
	buildIt := func(eventName string) {
		err := buildForServer(config, time.Now())
		if err != nil {
			log.Println(err)
		} else {
			if serverStarted && eventName != "" {
				lr.Reload(eventName)
			}
		}
	}

	watcher, err := NewRecursiveWatcher(srcDir, []string{config.OutputDirectory}, buildIt)

	if err != nil {
		return err
	}
	defer watcher.Close()

	fileServer := http.FileServer(http.Dir(serveDirectory))
	mux := http.NewServeMux()
	mux.Handle("/", fileServer)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	buildIt("")

	go func() {
		log.Printf("Serving %s on HTTP port: %s\n", serveDirectory, port)
		serverStarted = true
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	<-s.done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		// extra handling here
		watcher.Close()
		log.Println("Cleaning up...")
		os.RemoveAll(serveDirectory)
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.done <- true
}
