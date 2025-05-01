package commands

import (
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/asartalo/assg/internal/server"
)

func Serve(srcDir string, includeDrafts bool, verbose bool) error {
	ready := make(chan bool)
	stopSignal := make(chan os.Signal, 1)
	errorChannel := make(chan error)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv, err := server.NewServer(srcDir, includeDrafts, verbose)
	if err != nil {
		return err
	}

	go func() {
		<-ready
		<-stopSignal
		srv.Stop()
	}()

	go func() {
		err := srv.Start(ready)
		errorChannel <- err
	}()

	return <-errorChannel
}
