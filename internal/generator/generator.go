package generator

import (
	"cmp"
	"fmt"
	htmltpl "html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

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

type IndexTemplateContent struct {
	TemplateContent
	Pages []TemplateContent
	Prev  string
	Next  string
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

		_, err := g.GeneratePage(page, outputDir, *hierarchy, includeDrafts)
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

	destinationDir := path.Join(outputDir, page.RenderedPath())
	err = os.MkdirAll(destinationDir, 0755)
	if err != nil {
		return
	}

	templateData := pageToTemplateContent(page)

	if page.IsIndex() {
		pagingGroups := pagesToTemplateContents(page, hierarchy)
		pagingCount := len(pagingGroups)

		// render redirect page
		if pagingCount > 1 {
			page1Dir := path.Join(destinationDir, "page", "1")
			destinationPath = path.Join(page1Dir, "index.html")

			err = os.MkdirAll(page1Dir, 0755)
			if err != nil {
				return
			}
			redirectPath := g.FullUrl(page.RootPath())

			destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				return destinationPath, err
			}

			defer destinationFile.Close()
			err = g.Tmpl.RenderTemplate(
				"_redirect",
				destinationFile,
				redirectPath,
			)
		}

		for i, group := range pagingGroups {
			if i == 0 {
				destinationPath = path.Join(destinationDir, "index.html")
			} else {
				newDir := path.Join(destinationDir, "page", strconv.Itoa(i+1))
				destinationPath = path.Join(newDir, "index.html")

				err = os.MkdirAll(newDir, 0755)
				if err != nil {
					return
				}
			}

			prev := ""
			if i > 0 {
				if i == 1 {
					prev = slashPath(page.RootPath())
				} else {
					prev = slashPath(path.Join(page.RootPath(), "page", strconv.Itoa(i)))
				}
			}

			next := ""
			if pagingCount > 1 {
				lastIndex := pagingCount - 1
				if i < lastIndex {
					next = slashPath(path.Join(page.RootPath(), "page", strconv.Itoa(i+2)))
				}
			}

			indexTemplateData := IndexTemplateContent{
				TemplateContent: templateData,
				Pages:           group,
				Prev:            prev,
				Next:            next,
			}

			destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				return destinationPath, err
			}
			defer destinationFile.Close()

			err = g.Tmpl.RenderTemplate(
				templateToUse,
				destinationFile,
				indexTemplateData,
			)

			if err != nil {
				return destinationPath, err
			}

			err = TidyHtml(destinationPath)
			if err != nil {
				return destinationPath, err
			}

		}
	} else {
		destinationPath = path.Join(destinationDir, "index.html")
		// open destinationPath for writing
		destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return destinationPath, err
		}
		defer destinationFile.Close()
		//
		err = g.Tmpl.RenderTemplate(
			templateToUse,
			destinationFile,
			templateData,
		)

		if err != nil {
			return destinationPath, err
		}

		err = TidyHtml(destinationPath)
		if err != nil {
			return destinationPath, err
		}
	}

	return
}

func pageToTemplateContent(page *Page) TemplateContent {
	return TemplateContent{
		PageFrontMatter: page.FrontMatter,
		Content:         htmltpl.HTML(string(page.Content.String())),
	}
}

func pagesToTemplateContents(indexPage *Page, hierarchy PageHierarchy) [][]TemplateContent {
	childPages := hierarchy.GetChildren(*indexPage)
	// sort by  date
	slices.SortStableFunc(childPages, func(a, b *Page) int {
		return cmp.Compare(b.DateUnixEpoch(), a.DateUnixEpoch())
	})

	paginateBy := indexPage.FrontMatter.Index.PaginateBy
	return PaginateTransform(childPages, paginateBy, pageToTemplateContent)
}

func (g *Generator) GetTemplateToUse(page *Page, hierarchy PageHierarchy) string {
	templateToUse := DEFAULT_TEMPLATE
	parent := hierarchy.GetParent(*page)

	if page.FrontMatter.Template != "" {
		templateToUse = page.FrontMatter.Template
	} else if parent != nil && parent.FrontMatter.Index.PageTemplate != "" {
		templateToUse = parent.FrontMatter.Index.PageTemplate
	}

	return templateToUse
}

func (g *Generator) FullUrl(path string) string {
	return strings.TrimRight(g.Config.BaseURL, "/") + "/" + strings.TrimLeft(filepath.ToSlash(path), "/")
}

func slashPath(path string) string {
	tmp := filepath.ToSlash(path)
	length := len(tmp)
	if length > 0 && tmp[length-1] != '/' {
		return tmp + "/"
	}

	return tmp
}

func isMarkdown(info fs.DirEntry) bool {
	return filepath.Ext(info.Name()) == ".md"
}
