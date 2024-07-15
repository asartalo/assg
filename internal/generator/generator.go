package generator

import (
	"fmt"
	htmltpl "html/template"
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
	Content htmltpl.HTML
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
	err := g.Tmpl.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return err
	}

	contentDir := path.Join(srcDir, "content")

	err = filepath.WalkDir(contentDir, func(dPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if isMarkdown(info) {
				relPath, err := filepath.Rel(contentDir, dPath)
				if err != nil {
					return err
				}

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
			}
		}

		return nil
	})

	return err
}

func (g *Generator) GeneratePage(page *Page, outputDir string, includeDrafts bool) (destinationPath string, err error) {
	if page.FrontMatter.Draft && !includeDrafts {
		return "", nil
	}

	templateToUse := "default.html"
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
		Content: htmltpl.HTML(
			fmt.Sprintf("<h1>%s</h1>", page.FrontMatter.Title) + string(page.Content.String()),
		),
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
