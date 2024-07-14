package commands

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/generator"
)

type TemplateContent struct {
	generator.PageFrontMatter
	Content template.HTML
}

func Build(srcDir, outputDir string, includeDrafts bool) error {
	config, err := config.Load(path.Join(srcDir, "config.toml"))

	if err != nil {
		return err
	}

	gen := generator.New(config)
	err = gen.Tmpl.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return err
	}

	contentDir := path.Join(srcDir, "content")
	pathsToTidy := []string{}

	err = filepath.WalkDir(contentDir, func(dPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if isMarkdown(info) {
				renderedHtml, err := func() (destinationPath string, err error) {
					relPath, err := filepath.Rel(contentDir, dPath)
					if err != nil {
						return
					}

					fileContent, err := os.ReadFile(dPath)
					if err != nil {
						return
					}

					page, err := generator.ParsePage(relPath, fileContent)
					if err != nil {
						return
					}

					if page.FrontMatter.Draft && !includeDrafts {
						return "", nil
					}

					templateToUse := "default.html"
					if page.FrontMatter.Template != "" {
						templateToUse = page.FrontMatter.Template
					}

					destinationPath = path.Join(
						outputDir,
						getRenderedPath(relPath),
						"index.html",
					)
					pathsToTidy = append(pathsToTidy, destinationPath)

					destinationDir := filepath.Dir(destinationPath)
					err = os.MkdirAll(destinationDir, 0755)
					if err != nil {
						return
					}

					// open destinationPath for writing
					destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
					if err != nil {
						return
					}
					defer destinationFile.Close()

					templateData := TemplateContent{
						PageFrontMatter: page.FrontMatter,
						Content: template.HTML(
							fmt.Sprintf("<h1>%s</h1>", page.FrontMatter.Title) + string(page.Content.String()),
						),
					}

					err = gen.Tmpl.Templates.ExecuteTemplate(
						destinationFile,
						templateToUse,
						templateData,
					)

					return
				}()

				if err != nil {
					return err
				}

				err = tidyHtml(renderedHtml)

				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return err
}

var tidyArgs = []string{
	"--show-body-only",
	"auto",
	"--show-errors",
	"0",
	"--gnu-emacs",
	"yes",
	"-q",
	"-i",
	"-m",
	"-w",
	"160",
	"--indent-spaces",
	"2",
	"-ashtml",
	"-utf8",
	"--tidy-mark",
	"no",
}

func tidyHtml(pathToTidy string) error {
	args := append(tidyArgs, pathToTidy)
	cmd := exec.Command("tidy", args...)
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func isMarkdown(info fs.DirEntry) bool {
	return filepath.Ext(info.Name()) == ".md"
}

func getRenderedPath(relPath string) string {
	extension := filepath.Ext(relPath)
	lastDotIndex := len(relPath) - len(extension)
	return relPath[:lastDotIndex]
}
