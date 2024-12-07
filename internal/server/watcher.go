package server

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
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
