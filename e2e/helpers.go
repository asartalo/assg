package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"path"
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

func writeToFile(path string, content []byte) error {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	_, err = file.Write(content)

	return err
}

var tmpDir string

func initializeDirs() error {
	if tmpDir == "" {
		currentDir, err := filepath.Abs("../")
		if err != nil {
			return err
		}
		tmpDir = path.Join(currentDir, "tmp")
		return os.RemoveAll(path.Join(tmpDir, "diff"))
	}

	return nil
}

func writeDiffableFiles(expectedDir string, relPath string, expectedContent []byte, actualContent []byte) error {
	err := initializeDirs()
	if err != nil {
		return err
	}

	namespace := path.Base(path.Dir(expectedDir))
	nsDir := path.Join(tmpDir, "diff", namespace)
	expect := path.Join(nsDir, "expected", path.Dir(relPath))
	actual := path.Join(nsDir, "actual", path.Dir(relPath))
	err = os.MkdirAll(expect, 0755)
	if err != nil {
		return err
	}

	err = os.MkdirAll(actual, 0755)
	if err != nil {
		return err
	}
	err = writeToFile(path.Join(expect, path.Base(relPath)), expectedContent)
	if err != nil {
		return err
	}

	return writeToFile(path.Join(actual, path.Base(relPath)), actualContent)
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
				expectedStr := string(expectedContent)
				actualStr := string(actualContent)
				if expectedStr != actualStr {
					err := writeDiffableFiles(
						expectedDir,
						relPath,
						expectedContent,
						actualContent,
					)
					if err != nil {
						fmt.Println(err)
					}
				}

				assert.Equal(
					t,
					expectedStr,
					actualStr,
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
