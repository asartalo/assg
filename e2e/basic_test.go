package main

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

var knownTextFileExtensions = []string{".html", ".css", ".js", ".txt"}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func assertDirContents(t *testing.T, expectedDir string, actualDir string) {
	err := filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			relPath, err := filepath.Rel(expectedDir, path) // Get path relative to expectedDir
			if err != nil {
				return err
			}

			actualFilePath := filepath.Join(actualDir, relPath)

			// Assert file existence in actualDir
			assert.FileExists(
				t,
				actualFilePath,
				"File ./%s missing from actualDir",
				relPath,
			)

			// Assert file content equality
			expectedContent, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			actualContent, err := os.ReadFile(actualFilePath)
			if err != nil {
				return err
			}

			if contains(knownTextFileExtensions, filepath.Ext(path)) {
				assert.Equal(
					t,
					string(expectedContent),
					string(actualContent),
					"Content mismatch for file %s",
					relPath,
				)
			} else {
				assert.Equal(t, expectedContent, actualContent, "Content mismatch for file %s", relPath)
			}
		}

		return nil
	})

	assert.NoError(t, err, "Error during directory comparison")
}
