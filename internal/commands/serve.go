package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/generator"
	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
	"github.com/jaschaephraim/lrserver"
)

type RecursiveWatcher struct {
	watcher    *fsnotify.Watcher
	callback   func(name string)
	directory  string
	ignoreList map[string]bool
}

var commonIgnoreList = []string{
	".git",
	".sass-cache",
	"node_modules",
}

func NewRecursiveWatcher(directory string, ignore []string, callback func(name string)) (*RecursiveWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rWatcher := &RecursiveWatcher{
		watcher:    watcher,
		callback:   callback,
		directory:  directory,
		ignoreList: make(map[string]bool),
	}

	for _, toIgnore := range commonIgnoreList {
		rWatcher.ignoreList[toIgnore] = true
	}

	for _, toIgnore := range ignore {
		rWatcher.ignoreList[toIgnore] = true
	}

	rWatcher.watchDirectory(directory)
	rWatcher.start()

	return rWatcher, nil
}

func (r *RecursiveWatcher) watchDirectory(directory string) {
	r.watcher.Add(directory)
	file, err := os.Open(directory)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	files, err := file.Readdir(-1)
	if err != nil {
		log.Println(err)
		return
	}

	for _, f := range files {
		if f.IsDir() {
			if r.ignoreList[f.Name()] {
				continue
			}
			r.watchDirectory(path.Join(directory, f.Name()))
		}
	}
}

func (r *RecursiveWatcher) start() {
	go func() {
		debouncer := debounce.New(100 * time.Millisecond)
		debouncedCallback := func(eventName string) {
			debouncer(func() {
				r.callback(eventName)
			})
		}

		for {
			select {
			case event := <-r.watcher.Events:
				s, err := os.Stat(event.Name)
				if err == nil && s != nil && s.IsDir() {
					if event.Op&fsnotify.Create == fsnotify.Create {
						r.watchDirectory(event.Name)
					}
				}

				if event.Op&fsnotify.Remove == fsnotify.Remove {
					r.watcher.Remove(event.Name)
					debouncedCallback(event.Name)
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					debouncedCallback(event.Name)
				}

			case err := <-r.watcher.Errors:
				if err != nil {
					log.Println("error:", err)
				}
			}
		}
	}()
}

func (r *RecursiveWatcher) Close() {
	r.watcher.Close()
}

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

func Serve(srcDir string, includeDrafts bool) error {
	port := "8080"
	serveDirectory, err := os.MkdirTemp("", "public-assg")
	if err != nil {
		return err
	}

	config, err := config.Load(path.Join(srcDir, "config.toml"))
	if err != nil {
		return err
	}

	config.DevMode = true
	config.OutputDirectory = serveDirectory
	config.IncludeDrafts = includeDrafts
	config.BaseURL = fmt.Sprintf("http://localhost:%s", port)

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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Build the site for the first time
	buildIt("")

	go func() {
		log.Printf("Serving %s on HTTP port: %s\n", serveDirectory, port)
		serverStarted = true
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	<-done
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
