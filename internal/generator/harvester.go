package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/content"
)

func GatherContent(config config.Config, verbose bool) (*ContentHierarchy, error) {
	harvester := &harvester{
		outputDir:  config.OutputDirectoryAbsolute(),
		contentDir: config.ContentDirectoryAbsolute(),
		hierarchy:  NewPageHierarchy(verbose),
		verbose:    verbose,
	}

	return harvester.harvest()
}

type harvester struct {
	outputDir  string
	contentDir string
	hierarchy  *ContentHierarchy
	verbose    bool
}

func (harvester *harvester) harvest() (*ContentHierarchy, error) {
	hierarchy := harvester.hierarchy

	err := filepath.WalkDir(harvester.contentDir, func(contentPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// skip if basename has a dot prefix
			if filepath.Base(contentPath)[0] == '.' {
				return filepath.SkipDir
			}
		} else {
			relPath, err := filepath.Rel(harvester.contentDir, contentPath)
			if err != nil {
				return err
			}

			if isMarkdown(info) {
				harvester.Println("Processing markdown file:", relPath)
				return harvester.handleMarkdownFile(contentPath, relPath)
			} else {
				hierarchy.AddStaticFile(relPath, contentPath)
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	hierarchy.Retree()

	return hierarchy, err
}

func (h *harvester) Println(args ...interface{}) {
	if h.verbose {
		fmt.Println(args...)
	}
}

func (h *harvester) Printf(format string, args ...interface{}) {
	if h.verbose {
		fmt.Printf(format, args...)
	}
}

func (harvester *harvester) handleMarkdownFile(dPath string, relPath string) error {
	fileContent, err := os.ReadFile(dPath)
	if err != nil {
		return err
	}

	page, err := content.ParsePage(relPath, fileContent)
	if err != nil {
		return err
	}

	harvester.hierarchy.AddPage(page)

	return nil
}
