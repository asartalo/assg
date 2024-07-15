package e2e

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/amit7itz/goset"
	"github.com/stretchr/testify/assert"
)

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
	filesInExpectedDir := goset.NewSet[string]()
	filesInActualDir := goset.NewSet[string]()
	directoriesInExpectedDir := goset.NewSet[string]()
	directoriesInActualDir := goset.NewSet[string]()

	err := filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			directoriesInExpectedDir.Add(path)
		} else {
			filesInExpectedDir.Add(path)
			relPath, err := filepath.Rel(expectedDir, path) // Get path relative to expectedDir
			if err != nil {
				return err
			}

			actualFilePath := filepath.Join(actualDir, relPath)

			// Assert file existence in actualDir
			assert.FileExists(
				t,
				actualFilePath,
				"Expected file ./%s missing in output directory",
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

	assert.NoError(t, err, "Error walking through expected directory")

	err = filepath.WalkDir(actualDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			directoriesInActualDir.Add(path)
		} else {
			filesInActualDir.Add(path)
		}

		return nil
	})

	assert.NoError(t, err, "Error tallying output directory")

	if directoriesInActualDir.Len() > directoriesInExpectedDir.Len() {
		t.Errorf(
			"Extra directories in output directory: %v",
			directoriesInActualDir.Difference(directoriesInExpectedDir),
		)
	}

	if filesInActualDir.Len() > filesInExpectedDir.Len() {
		t.Errorf(
			"Extra files in output directory: %v",
			filesInActualDir.Difference(filesInExpectedDir),
		)
	}
}
