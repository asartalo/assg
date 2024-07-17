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

func (g *Generator) Build(srcDir, outputDir string, includeDrafts bool) error {
	err := g.Tmpl.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return err
	}

	contentDir := path.Join(srcDir, "content")
	hierarchy := NewPageHierarchy()

	err = filepath.WalkDir(contentDir, func(dPath string, info fs.DirEntry, err error) error {
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

				hierarchy.AddPage(page)
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

	hierarchy.Retree()

	for _, node := range hierarchy.Pages {
		page := node.Page

		renderedHtml, err := g.GeneratePage(page, outputDir, *hierarchy, includeDrafts)
		if err != nil {
			return err
		}

		err = TidyHtml(renderedHtml)
		if err != nil {
			return err
		}
	}

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

const DEFAULT_TEMPLATE = "default.html"

func (g *Generator) GeneratePage(page *Page, outputDir string, hierarchy PageHierarchy, includeDrafts bool) (destinationPath string, err error) {
	if page.FrontMatter.Draft && !includeDrafts {
		return "", nil
	}

	templateToUse := g.GetTemplateToUse(page, hierarchy)

	// check if template is defined
	if !g.Tmpl.TemplateExists(templateToUse) {
		return "", fmt.Errorf(
			"the template \"%s\" for the page \"%s\" does not exist",
			templateToUse,
			page.Path,
		)
	}

	destinationPath = path.Join(
		outputDir,
		page.RenderedPath(),
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
		Content:         htmltpl.HTML(string(page.Content.String())),
	}

	err = g.Tmpl.RenderTemplate(
		templateToUse,
		destinationFile,
		templateData,
	)

	return
}

func (g *Generator) GetTemplateToUse(page *Page, hierarchy PageHierarchy) string {
	templateToUse := DEFAULT_TEMPLATE
	parent := hierarchy.GetParent(*page)
	// print hierarchy.Pages keys
	for k := range hierarchy.Pages {
		fmt.Println(k)
	}
	if page.FrontMatter.Template != "" {
		templateToUse = page.FrontMatter.Template
	} else if parent != nil && parent.FrontMatter.Index.PageTemplate != "" {
		templateToUse = parent.FrontMatter.Index.PageTemplate
	}

	return templateToUse
}

func isMarkdown(info fs.DirEntry) bool {
	return filepath.Ext(info.Name()) == ".md"
}
