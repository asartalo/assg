package generator

import (
	"bytes"
	"fmt"
	htmltpl "html/template"
	"os"
	"path"
	"strconv"
	"time"

	"codeberg.org/asartalo/assg/internal/config"
	"codeberg.org/asartalo/assg/internal/content"
	"github.com/gertd/go-pluralize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type PageGenerator struct {
	mg        *Generator
	Config    *config.Config
	hierarchy *ContentHierarchy
}

func (pg *PageGenerator) Printf(format string, args ...any) {
	pg.mg.Printf(format, args...)
}

func (pg *PageGenerator) GeneratePage(page *content.WebPage, now time.Time) (err error) {
	if page == nil {
		return fmt.Errorf("page is nil")
	}

	pg.Printf("Generating page: %s\n", page.MarkdownPath)

	if page.FrontMatter.Draft && !pg.Config.IncludeDrafts {
		return nil
	}

	templateToUse := pg.GetTemplateToUse(page)

	// check if template is defined
	if !pg.mg.Tmpl.TemplateExists(templateToUse) {
		return fmt.Errorf(
			"the template \"%s\" for the page \"%s\" does not exist",
			templateToUse,
			page.MarkdownPath,
		)
	}

	pg.Printf("  Using template: %s\n", templateToUse)

	pagePath := page.RenderedPath()
	templateData := pg.PageToTemplateContent(page)
	pg.Printf("  Destination: %s\n", pagePath)

	if page.IsTaxonomy() {
		err = pg.generateTaxonomyPages(
			page,
			templateData,
			pagePath,
			templateToUse,
			now,
		)
	} else if page.IsIndex() {
		err = pg.generateIndexPages(
			page,
			templateData,
			pagePath,
			templateToUse,
			pg.PagesToTemplateContents(page),
		)
	} else {
		parentPage := pg.hierarchy.GetParent(*page)
		if parentPage != nil {
			pg.Printf("  Parent page: %s\n", parentPage.MarkdownPath)
			err = pg.renderPage(
				pg.generateChildPageData(page, parentPage, templateData),
				pagePath,
				templateToUse,
				true,
			)
		} else {
			pg.Printf("  Not a child page\n")
			err = pg.renderPage(templateData, pagePath, templateToUse, true)
		}
	}

	return
}

func (pg *PageGenerator) GetTemplateToUse(page *content.WebPage) string {
	templateToUse := DEFAULT_TEMPLATE
	parent := pg.hierarchy.GetParent(*page)

	if page.FrontMatter.Template != "" {
		templateToUse = page.FrontMatter.Template
	} else if parent != nil && parent.FrontMatter.Index.PageTemplate != "" {
		templateToUse = parent.FrontMatter.Index.PageTemplate
	}

	return templateToUse
}

func (pg *PageGenerator) PageToTemplateContent(page *content.WebPage) TemplateContent {
	summary := ""
	actualSummary, err := page.Summary()
	if err == nil {
		summary = actualSummary
	}

	return TemplateContent{
		FrontMatter: page.FrontMatter,
		Content:     htmltpl.HTML(string(page.Content.String())),
		Config:      *pg.Config,
		RootPath:    page.RootPath(),
		Permalink:   pg.mg.FullUrl(page.RootPath()),
		Path:        page.RenderedPath(),
		Summary:     htmltpl.HTML(summary),
	}
}

func (pg *PageGenerator) PagesToTemplateContents(indexPage *content.WebPage) [][]TemplateContent {
	childPages := pg.hierarchy.GetChildren(*indexPage)

	paginateBy := indexPage.FrontMatter.Index.PaginateBy
	return PaginateTransform(childPages, paginateBy, pg.PageToTemplateContent)
}

func (pg *PageGenerator) GetSectionPages(indexPath string, max int, offset int) (sectionPages []TemplateContent) {
	section := pg.hierarchy.GetPage(indexPath)
	if section == nil {
		return sectionPages
	}

	children := pg.hierarchy.GetChildren(*section)
	for i, page := range children {
		if i < offset {
			continue
		}

		if i >= offset+max {
			break
		}

		sectionPages = append(sectionPages, pg.PageToTemplateContent(page))
	}

	return sectionPages
}

func (pg *PageGenerator) generateTaxonomyPages(
	page *content.WebPage,
	templateData TemplateContent,
	pagePath string,
	templateToUse string,
	now time.Time,
) (err error) {
	pg.Printf("  Generating taxonomy pages for: %s\n", page.MarkdownPath)
	taxonomy := page.TaxonomyType()
	termMapping := pg.hierarchy.GetTaxonomyTerms(taxonomy)
	paginateBy := page.FrontMatter.Index.PaginateBy

	err = pg.renderPage(templateData, pagePath, templateToUse, true)
	if err != nil {
		return err
	}

	titleCaser := cases.Title(language.English).String
	indexTemplateToUse := page.FrontMatter.Index.PageTemplate
	pluralizer := pluralize.NewClient()
	taxonomySingular := pluralizer.Singular(titleCaser(taxonomy))
	for term, pages := range termMapping {
		termDir := path.Join(pagePath, dashSpaces(term))
		taxIndexFields := page.FrontMatter.Index
		iPageFrontMatter := content.FrontMatter{
			Title: titleCaser(term),
			Date:  now,
			Description: fmt.Sprintf(
				"%s: %s",
				taxonomySingular,
				titleCaser(term),
			),
			Index:    taxIndexFields,
			Template: page.FrontMatter.Index.PageTemplate,
		}
		termPage := content.WebPage{
			FrontMatter: iPageFrontMatter,
			Content:     *new(bytes.Buffer),
			MarkdownPath: path.Join(
				page.RenderedPath(),
				fmt.Sprintf("%s.md", dashSpaces(term)),
			),
		}

		err = pg.generateIndexPages(
			&termPage,
			pg.PageToTemplateContent(&termPage),
			termDir,
			indexTemplateToUse,
			PaginateTransform(pages, paginateBy, pg.PageToTemplateContent),
		)

		if err != nil {
			return err
		}
	}

	return
}

func (pg *PageGenerator) generateChildPageData(
	page *content.WebPage,
	parentPage *content.WebPage,
	templateData TemplateContent,
) PaginatedTemplateContent {

	prev := ""
	prevPage := pg.hierarchy.GetPrevPage(parentPage, page)
	var prevPageData TemplateContent
	if prevPage != nil {
		prev = prevPage.RootPath()
		prevPageData = pg.PageToTemplateContent(prevPage)
	}

	next := ""
	nextPage := pg.hierarchy.GetNextPage(parentPage, page)
	var nextPageData TemplateContent
	if nextPage != nil {
		next = nextPage.RootPath()
		nextPageData = pg.PageToTemplateContent(nextPage)
	}

	return PaginatedTemplateContent{
		TemplateContent: templateData,
		Prev:            prev,
		PrevPage:        prevPageData,
		Next:            next,
		NextPage:        nextPageData,
	}
}

func (pg *PageGenerator) renderPage(
	templateData any,
	pagePath string,
	templateToUse string,
	canonical bool,
) error {
	g := pg.mg
	destinationDir := g.OutputPath(pagePath)
	err := os.MkdirAll(destinationDir, 0755)
	if err != nil {
		return err
	}

	pg.Printf("  Rendering page: %s\n", g.pathValue(templateData))
	destinationPath := path.Join(destinationDir, "index.html")
	destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		pg.Printf("Error creating file")
		return err
	}

	err = g.Tmpl.RenderTemplate(
		templateToUse,
		destinationFile,
		templateData,
	)

	if err != nil {
		pg.Printf("Error rendering %s\n", destinationPath)
		return err
	}

	err = destinationFile.Close()
	if err != nil {
		pg.Printf("Error closing file %s\n", destinationPath)
		return err
	}

	if canonical {
		g.renderedPaths = append(g.renderedPaths, content.RootPath(pagePath))
	}

	return err
}

func (pg *PageGenerator) generateIndexPages(
	page *content.WebPage,
	templateData TemplateContent,
	pagePath string,
	templateToUse string,
	pagingGroups [][]TemplateContent,
) (err error) {
	g := pg.mg
	pg.Printf("  Generating index pages for: %s\n", page.MarkdownPath)
	pagingCount := len(pagingGroups)

	// render redirect page
	if pagingCount > 1 {
		page1Path := path.Join(pagePath, "page", "1")
		redirectPath := g.FullUrl(page.RootPath())

		err = pg.renderPage(redirectPath, page1Path, "_redirect", false)
		if err != nil {
			return err
		}
	}

	for i, group := range pagingGroups {
		var destinPath string
		if i == 0 {
			destinPath = pagePath
		} else {
			destinPath = path.Join(pagePath, "page", strconv.Itoa(i+1))
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
			CurrentPage:     i + 1,
			TotalPages:      pagingCount,
		}

		err = pg.renderPage(indexTemplateData, destinPath, templateToUse, true)
	}

	return
}
