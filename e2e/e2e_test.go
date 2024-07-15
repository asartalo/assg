package e2e

import (
	"os"
	"path"
	"testing"

	"github.com/asartalo/assg/internal/commands"
	"github.com/stretchr/testify/assert"
)

func TestBasicSite(t *testing.T) {
	// Setup test environment
	publicDir, err := os.MkdirTemp("", "basic-public")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(publicDir)

	// current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working directory")
	}

	siteDir := path.Join(cwd, "fixtures", "basic")
	expectedDir := path.Join(siteDir, "public-expected")

	err = commands.Build(siteDir, publicDir, false)
	assert.NoError(t, err)

	assertDirContents(t, expectedDir, publicDir)
}

func TestSiteHomeOnly(t *testing.T) {
	// Setup test environment
	publicDir, err := os.MkdirTemp("", "site-home-only-public")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(publicDir)

	// current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working directory")
	}

	siteDir := path.Join(cwd, "fixtures", "site-home-only")
	expectedDir := path.Join(siteDir, "public-expected")

	err = commands.Build(siteDir, publicDir, false)
	assert.NoError(t, err)

	assertDirContents(t, expectedDir, publicDir)
}
