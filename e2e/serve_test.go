package e2e

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"github.com/asartalo/assg/internal/server"
	"github.com/chromedp/chromedp"
)

func TestIt(t *testing.T) {
	t.Parallel()

	// chromedp.WithLogf(t.Logf)
	c := context.Background()

	// Create a new browser
	ctx, cancel := chromedp.NewContext(
		c,
		chromedp.WithLogf(t.Logf),
	)
	defer cancel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixturesDirectory := path.Join(cwd, "fixtures")

	// Start the test server
	srv, err := server.NewServer(path.Join(fixturesDirectory, "blog-posts"), false)
	if err != nil {
		t.Fatal(err)
	}

	ready := make(chan bool)
	go srv.Start(ready)
	log.Println("Server started")
	defer srv.Stop()
	<-ready

	log.Println("Checking the site")
	// Run the browser
	var result string
	resp, err := chromedp.RunResponse(ctx,
		chromedp.Navigate("http://localhost:8181/"),
	)

	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != 200 {
		t.Fatalf("got unexpected status code: %d", resp.Status)
	}

	chromedp.Run(ctx, chromedp.Title(&result))

	// Check the result
	if result != "My Blog" {
		t.Errorf("got unexpected title: %q", result)
	}
}
