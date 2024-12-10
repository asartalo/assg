package e2e

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/amit7itz/goset"
	"github.com/stretchr/testify/assert"
)

var knownTextFileExtensions = []string{".html", ".css", ".js", ".txt", ".xml"}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func gatherFilesAndDirectoriesInDir(dir string) (files *goset.Set[string], directories *goset.Set[string], err error) {
	files = goset.NewSet[string]()
	directories = goset.NewSet[string]()

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			directories.Add(path)
		} else {
			files.Add(path)
		}

		return nil
	})

	return
}

func compareExpectedDirContentsToActual(t *testing.T, expectedDir, actualDir string) error {
	return filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
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
}

func assertDirContents(t *testing.T, expectedDir, actualDir string) {
	filesInExpectedDir, directoriesInExpectedDir, err := gatherFilesAndDirectoriesInDir(expectedDir)
	assert.NoError(t, err, "Error comparinng expected directory with actual")

	err = compareExpectedDirContentsToActual(t, expectedDir, actualDir)
	assert.NoError(t, err, "Error tallying expected directory")

	filesInActualDir, directoriesInActualDir, err := gatherFilesAndDirectoriesInDir(actualDir)
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
