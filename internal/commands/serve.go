package commands

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/asartalo/assg/internal/server"
)

func Serve(srcDir string, includeDrafts bool) error {
	serverSignal := make(chan os.Signal, 1)
	signal.Notify(serverSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv, err := server.NewServer(srcDir, includeDrafts)
	if err != nil {
		return err
	}

	go func() {
		<-serverSignal
		srv.Stop()
	}()

	return srv.Start()
}
