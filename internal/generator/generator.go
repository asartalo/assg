package generator

import (
	"fmt"
	htmltpl "html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/template"
)

type Generator struct {
	Config    *config.Config
	Tmpl      *template.Engine
	hierarchy *ContentHierarchy
}

func New(cfg *config.Config) (*Generator, error) {
	generator := &Generator{
		Config: cfg,
	}

	hierarchy, err := GatherContent(*cfg)
	if err != nil {
		return nil, err
	}

	generator.hierarchy = hierarchy

	funcMap := make(htmltpl.FuncMap)
	funcMap["sectionPages"] = func(indexPath string, max, offset int) []TemplateContent {
		return generator.GetSectionPages(indexPath, max, offset)
	}

	funcMap["sectionIndex"] = func(indexPath string) IndexTemplateContent {
		page := generator.hierarchy.GetPage(indexPath)
		if page == nil {
			fmt.Printf("Unable to find page \"%s\"\n", indexPath)
		}
		templateContent := generator.PageToTemplateContent(page)

		return IndexTemplateContent{
			TemplateContent: templateContent,
		}
	}

	srcDir := cfg.RootDirectory()
	templates := template.New(funcMap)
	err = templates.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return nil, err
	}

	generator.Tmpl = templates

	return generator, err
}

func (g *Generator) Build() error {
	for _, node := range g.hierarchy.Pages {
		err := g.GeneratePage(node.Page)
		if err != nil {
			return err
		}
	}

	return nil
}

func GatherContent(config config.Config) (*ContentHierarchy, error) {
	outputDir := config.OutputDirectoryAbsolute()
	contentDir := config.ContentDirectoryAbsolute()
	hierarchy := NewPageHierarchy()

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

	if err != nil {
		return nil, err
	}

	hierarchy.Retree()

	return hierarchy, err
}

const DEFAULT_TEMPLATE = "default.html"

func (g *Generator) GeneratePage(page *WebPage) (err error) {
	if page == nil {
		return fmt.Errorf("page is nil")
	}

	if page.FrontMatter.Draft && !g.Config.IncludeDrafts {
		return nil
	}

	templateToUse := g.GetTemplateToUse(page)

	// check if template is defined
	if !g.Tmpl.TemplateExists(templateToUse) {
		return fmt.Errorf(
			"the template \"%s\" for the page \"%s\" does not exist",
			templateToUse,
			page.Path,
		)
	}

	destinationDir := path.Join(g.Config.OutputDirectoryAbsolute(), page.RenderedPath())
	templateData := g.PageToTemplateContent(page)

	if page.IsIndex() {
		err = g.generateIndexPages(
			page,
			templateData,
			destinationDir,
			templateToUse,
		)
	} else {
		parentPage := g.hierarchy.GetParent(*page)
		if parentPage != nil {
			err = g.renderPage(
				g.generateChildPageData(page, parentPage, templateData),
				destinationDir,
				templateToUse,
			)
		} else {
			err = g.renderPage(templateData, destinationDir, templateToUse)
		}
	}

	return
}

func (g *Generator) GetSectionPages(indexPath string, max int, offset int) (sectionPages []TemplateContent) {
	section := g.hierarchy.GetPage(indexPath)
	if section == nil {
		return sectionPages
	}

	children := g.hierarchy.GetChildren(*section)
	for i, page := range children {
		if i < offset {
			continue
		}

		if i >= offset+max {
			break
		}

		sectionPages = append(sectionPages, g.PageToTemplateContent(page))
	}

	return sectionPages
}

func (g *Generator) generateIndexPages(
	page *WebPage,
	templateData TemplateContent,
	destinationDir string,
	templateToUse string,
) (err error) {
	pagingGroups := g.PagesToTemplateContents(page)
	pagingCount := len(pagingGroups)

	// render redirect page
	if pagingCount > 1 {
		page1Dir := path.Join(destinationDir, "page", "1")
		redirectPath := g.FullUrl(page.RootPath())

		err = g.renderPage(redirectPath, page1Dir, "_redirect")
		if err != nil {
			return err
		}
	}

	for i, group := range pagingGroups {
		var destinDir string
		if i == 0 {
			destinDir = destinationDir
		} else {
			destinDir = path.Join(destinationDir, "page", strconv.Itoa(i+1))
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

		err = g.renderPage(indexTemplateData, destinDir, templateToUse)
	}

	return
}

func (g *Generator) generateChildPageData(
	page *WebPage,
	parentPage *WebPage,
	templateData TemplateContent,
) PaginatedTemplateContent {

	prev := ""
	prevPage := g.hierarchy.GetPrevPage(parentPage, page)
	var prevPageData TemplateContent
	if prevPage != nil {
		prev = prevPage.RootPath()
		prevPageData = g.PageToTemplateContent(prevPage)
	}

	next := ""
	nextPage := g.hierarchy.GetNextPage(parentPage, page)
	var nextPageData TemplateContent
	if nextPage != nil {
		next = nextPage.RootPath()
		nextPageData = g.PageToTemplateContent(nextPage)
	}

	return PaginatedTemplateContent{
		TemplateContent: templateData,
		Prev:            prev,
		PrevPage:        prevPageData,
		Next:            next,
		NextPage:        nextPageData,
	}
}

func (g *Generator) renderPage(templateData interface{}, destinationDir string, templateToUse string) error {
	err := os.MkdirAll(destinationDir, 0755)
	if err != nil {
		return err
	}

	destinationPath := path.Join(destinationDir, "index.html")
	destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		fmt.Printf("Error creating file")
		return err
	}
	defer destinationFile.Close()

	err = g.Tmpl.RenderTemplate(
		templateToUse,
		destinationFile,
		templateData,
	)
	if err != nil {
		return err
	}

	return TidyHtml(destinationPath)
}

func (g *Generator) GetTemplateToUse(page *WebPage) string {
	templateToUse := DEFAULT_TEMPLATE
	parent := g.hierarchy.GetParent(*page)

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

func (g *Generator) PageToTemplateContent(page *WebPage) TemplateContent {
	return TemplateContent{
		FrontMatter: page.FrontMatter,
		Content:     htmltpl.HTML(string(page.Content.String())),
		Config:      *g.Config,
		Permalink:   g.FullUrl(page.RootPath()),
	}
}

func (g *Generator) PagesToTemplateContents(indexPage *WebPage) [][]TemplateContent {
	childPages := g.hierarchy.GetChildren(*indexPage)

	paginateBy := indexPage.FrontMatter.Index.PaginateBy
	return PaginateTransform(childPages, paginateBy, g.PageToTemplateContent)
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
