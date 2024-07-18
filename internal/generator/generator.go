package generator

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/template"
)

type Generator struct {
	Config *config.Config
	Tmpl   *template.Engine
}

type TemplateContent struct {
	PageFrontMatter
	Content []byte
}

func New(cfg *config.Config) *Generator {
	return &Generator{
		Config: cfg,
		Tmpl:   template.New(),
	}
}

func getRenderedPath(relPath string) string {
	// if the file is named index.md, we want to render it as the root index.html (e.g. /index.html)
	if relPath == "index.md" {
		return ""
	}

	extension := filepath.Ext(relPath)
	lastDotIndex := len(relPath) - len(extension)
	return relPath[:lastDotIndex]
}

func (g *Generator) Build(srcDir, outputDir string, includeDrafts bool) error {
	g.Tmpl.LoadTemplates(path.Join(srcDir, "templates"))

	contentDir := path.Join(srcDir, "content")

	err := filepath.WalkDir(contentDir, func(dPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// skip if basename has a dot prefix
			if filepath.Base(dPath)[0] == '.' {
				return filepath.SkipDir
			}
		} else {
			relPath, err := filepath.Rel(contentDir, dPath)
			if err != nil {
				return err
			}

			if isMarkdown(info) {
				fileContent, err := os.ReadFile(dPath)
				if err != nil {
					return err
				}

				page, err := ParsePage(relPath, fileContent)
				if err != nil {
					return err
				}

				renderedHtml, err := g.GeneratePage(page, outputDir, includeDrafts)
				if err != nil {
					return err
				}

				err = TidyHtml(renderedHtml)
				if err != nil {
					return err
				}
			} else {
				// everything else is just copied over
				destinationPath := path.Join(outputDir, relPath)
				err = os.MkdirAll(filepath.Dir(destinationPath), 0755)
				if err != nil {
					return err
				}

				err = copyFile(dPath, destinationPath)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return err
}

func copyFile(from, to string) error {
	// open sourcefile for reading
	source, err := os.OpenFile(from, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer source.Close()

	// open destinationPath for writing
	destination, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer destination.Close()

	// copy file
	_, err = destination.ReadFrom(source)

	return err
}

const DEFAULT_TEMPLATE = "default.jet"

func (g *Generator) GeneratePage(page *Page, outputDir string, includeDrafts bool) (destinationPath string, err error) {
	if page.FrontMatter.Draft && !includeDrafts {
		return "", nil
	}

	templateToUse := DEFAULT_TEMPLATE
	if page.FrontMatter.Template != "" {
		templateToUse = page.FrontMatter.Template
	}

	destinationPath = path.Join(
		outputDir,
		getRenderedPath(page.Path),
		"index.html",
	)

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
		Content:         page.Content.Bytes(),
	}

	err = g.Tmpl.RenderTemplate(
		templateToUse,
		destinationFile,
		templateData,
	)

	return
}

func isMarkdown(info fs.DirEntry) bool {
	return filepath.Ext(info.Name()) == ".md"
}
