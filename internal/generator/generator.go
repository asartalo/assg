package generator

import (
	"bytes"
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
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/content"
	"github.com/asartalo/assg/internal/template"
	"github.com/gertd/go-pluralize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Generator struct {
	Config     *config.Config
	Tmpl       *template.Engine
	hierarchy  *ContentHierarchy
	feedAuthor *FeedAuthor
}

func defineFuncs(generator *Generator) htmltpl.FuncMap {
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

	funcMap["taxonomyTerms"] = func(taxonomy string) []TaxonomyTermContent {
		return generator.GetAllTaxonomyTerms(taxonomy)
	}

	funcMap["atomUrl"] = func() string {
		return generator.FullUrl("atom.xml")
	}

	funcMap["atomLink"] = func() htmltpl.HTML {
		return htmltpl.HTML(fmt.Sprintf(
			`<link rel="alternate" type="application/atom+xml" href="%s">`,
			generator.FullUrl("atom.xml"),
		))
	}

	funcMap["pageTaxonomy"] = func(path, taxonomy string) []TaxonomyTermContent {
		return generator.GetTaxonomyTermsForPage(path, taxonomy)
	}

	return funcMap
}

func New(cfg *config.Config) (*Generator, error) {
	srcDir := cfg.RootDirectory()

	generator := &Generator{
		Config: cfg,
	}

	hierarchy, err := GatherContent(*cfg)
	if err != nil {
		return nil, err
	}
	generator.hierarchy = hierarchy

	funcMap := defineFuncs(generator)
	templates := template.New(funcMap)
	err = templates.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return nil, err
	}

	generator.Tmpl = templates

	return generator, err
}

func (g *Generator) Build(now time.Time) error {
	for _, node := range g.hierarchy.Pages {
		err := g.GeneratePage(node.Page)
		if err != nil {
			return err
		}
	}

	g.GenerateFeed(now)

	return nil
}

func (g *Generator) defaultFeedAuthor() *FeedAuthor {
	if g.feedAuthor == nil {
		g.feedAuthor = &FeedAuthor{
			Name: g.Config.Author,
		}
	}

	return g.feedAuthor
}

func (g *Generator) GenerateFeed(now time.Time) error {
	if !g.Config.GenerateFeed {
		return nil
	}

	atomUrl := g.FullUrl("atom.xml")
	feed := Feed{
		Xmlns:     "http://www.w3.org/2005/Atom",
		Lang:      "en",
		Title:     g.Config.Title,
		Subtitle:  g.Config.Description,
		Id:        atomUrl,
		Generator: &FeedGenerator{Uri: "https://github.com/asartalo/assg", Name: "ASSG"},
		Updated:   FeedDateTime(now),
		Links: []*FeedLink{
			{
				Rel:  "self",
				Type: "application/atom+xml",
				Href: atomUrl,
			},
			{
				Rel:  "alternate",
				Type: "text/html",
				Href: g.SiteUrlNoTrailingslash(),
			},
		},
	}

	for _, page := range g.hierarchy.SortedPages() {
		if page.IsTaxonomy() || page.IsIndex() {
			continue
		}

		entry, err := g.createFeedEntry(page)
		if err != nil {
			return err
		}

		feed.Entries = append(feed.Entries, entry)
	}

	atomFilePath := g.OutputPath("atom.xml")

	atomFile, err := os.OpenFile(atomFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	err = feed.WriteXML(atomFile)
	if err != nil {
		return err
	}

	atomFile.Close()
	return TidyXml(atomFilePath)
}

func (g *Generator) createFeedEntry(page *content.WebPage) (*FeedEntry, error) {
	pageUrl := g.FullUrl(page.RootPath())

	item := &FeedEntry{
		Lang:  "en",
		Title: page.FrontMatter.Title,
		Links: []*FeedLink{
			{Rel: "alternate", Type: "text/html", Href: pageUrl},
		},
		Published: FeedDateTime(page.FrontMatter.Date),
		Updated:   FeedDateTime(page.FrontMatter.Date),
		Id:        pageUrl,
		Authors:   []*FeedAuthor{g.defaultFeedAuthor()},
	}

	// If the content is too short, use that instead of summary
	if page.Content.Len() > 500 {
		summary, err := page.Summary()
		if err != nil {
			return nil, err
		}

		item.Summary = &FeedEntrySummary{
			Type:    "html",
			Content: summary,
		}
	} else {
		item.Content = &FeedContent{
			Type:    "html",
			Base:    pageUrl,
			Content: strings.TrimSpace(page.Content.String()),
		}
	}

	return item, nil
}

func GatherContent(config config.Config) (*ContentHierarchy, error) {
	harvester := &harvester{
		outputDir:  config.OutputDirectoryAbsolute(),
		contentDir: config.ContentDirectoryAbsolute(),
		hierarchy:  NewPageHierarchy(),
	}

	return harvester.harvest()
}

type harvester struct {
	outputDir  string
	contentDir string
	hierarchy  *ContentHierarchy
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
				return harvester.handleMarkdownFile(contentPath, relPath)
			} else {
				return harvester.copyFiles(contentPath, relPath)
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

func (harvester *harvester) copyFiles(dPath string, relPath string) error {
	destinationPath := path.Join(harvester.outputDir, relPath)
	err := os.MkdirAll(filepath.Dir(destinationPath), 0755)
	if err != nil {
		return err
	}

	err = copyFile(dPath, destinationPath)
	if err != nil {
		return err
	}
	return err
}

const DEFAULT_TEMPLATE = "default.html"

func (g *Generator) OutputPath(endPath string) string {
	return path.Join(g.Config.OutputDirectoryAbsolute(), endPath)
}

func (g *Generator) GeneratePage(page *content.WebPage) (err error) {
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
			page.MarkdownPath,
		)
	}

	destinationDir := g.OutputPath(page.RenderedPath())
	templateData := g.PageToTemplateContent(page)
	if page.IsTaxonomy() {
		err = g.generateTaxonomyPages(
			page,
			templateData,
			destinationDir,
			templateToUse,
		)
	} else if page.IsIndex() {
		err = g.generateIndexPages(
			page,
			templateData,
			destinationDir,
			templateToUse,
			g.PagesToTemplateContents(page),
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

func (g *Generator) GetAllTaxonomyTerms(taxonomy string) (termTemplates []TaxonomyTermContent) {
	mapTerms := g.hierarchy.GetTaxonomyTerms(taxonomy)
	taxonomyIndexPage := g.hierarchy.GetTaxonomyPage(taxonomy)

	for term, pages := range mapTerms {
		rootPath := content.RootPath(filepath.ToSlash(path.Join(taxonomyIndexPage.RenderedPath(), term)))
		termTemplates = append(termTemplates, TaxonomyTermContent{
			Term:      term,
			PageCount: len(pages),
			RootPath:  rootPath,
			Permalink: g.FullUrl(rootPath),
		})
	}

	slices.SortStableFunc(termTemplates, func(a, b TaxonomyTermContent) int {
		return cmp.Compare(a.Term, b.Term)
	})
	return termTemplates
}

func (g *Generator) GetTaxonomyTermsForPage(rootPath string, taxonomy string) (termTemplates []TaxonomyTermContent) {
	ofPage := g.hierarchy.GetPage(rootPath)
	mapTerms := g.hierarchy.GetTaxonomyTerms(taxonomy)
	terms := ofPage.FrontMatter.Taxonomies[taxonomy]
	taxonomyIndexPage := g.hierarchy.GetTaxonomyPage(taxonomy)

	for _, term := range terms {
		pages := mapTerms[term]
		rootPath := content.RootPath(filepath.ToSlash(path.Join(taxonomyIndexPage.RenderedPath(), term)))
		termTemplates = append(termTemplates, TaxonomyTermContent{
			Term:      term,
			PageCount: len(pages),
			RootPath:  rootPath,
			Permalink: g.FullUrl(rootPath),
		})
	}

	slices.SortStableFunc(termTemplates, func(a, b TaxonomyTermContent) int {
		return cmp.Compare(a.Term, b.Term)
	})

	return termTemplates
}

func (g *Generator) generateTaxonomyPages(
	page *content.WebPage,
	templateData TemplateContent,
	destinationDir string,
	templateToUse string,
) (err error) {
	taxonomy := page.TaxonomyType()
	termMapping := g.hierarchy.GetTaxonomyTerms(taxonomy)
	paginateBy := page.FrontMatter.Index.PaginateBy

	err = g.renderPage(templateData, destinationDir, templateToUse)
	if err != nil {
		return err
	}

	now := time.Now()
	titleCaser := cases.Title(language.English).String
	indexTemplateToUse := page.FrontMatter.Index.PageTemplate
	pluralizer := pluralize.NewClient()
	taxonomySingular := pluralizer.Singular(titleCaser(taxonomy))
	for term, pages := range termMapping {
		termDir := path.Join(destinationDir, term)
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
			FrontMatter:  iPageFrontMatter,
			Content:      *new(bytes.Buffer),
			MarkdownPath: path.Join(page.RenderedPath(), fmt.Sprintf("%s.md", term)),
		}

		err = g.generateIndexPages(
			&termPage,
			g.PageToTemplateContent(&termPage),
			termDir,
			indexTemplateToUse,
			PaginateTransform(pages, paginateBy, g.PageToTemplateContent),
		)

		if err != nil {
			return err
		}
	}

	return
}

func (g *Generator) generateIndexPages(
	page *content.WebPage,
	templateData TemplateContent,
	destinationDir string,
	templateToUse string,
	pagingGroups [][]TemplateContent,
) (err error) {
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
	page *content.WebPage,
	parentPage *content.WebPage,
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

func (g *Generator) GetTemplateToUse(page *content.WebPage) string {
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
	return g.SiteUrlWithTrailingSlash() + strings.TrimLeft(filepath.ToSlash(path), "/")
}

func (g *Generator) SiteUrlNoTrailingslash() string {
	return strings.TrimRight(g.Config.BaseURL, "/")
}

func (g *Generator) SiteUrlWithTrailingSlash() string {
	return g.SiteUrlNoTrailingslash() + "/"
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

func (g *Generator) PageToTemplateContent(page *content.WebPage) TemplateContent {
	return TemplateContent{
		FrontMatter: page.FrontMatter,
		Content:     htmltpl.HTML(string(page.Content.String())),
		Config:      *g.Config,
		RootPath:    page.RootPath(),
		Permalink:   g.FullUrl(page.RootPath()),
		Path:        page.RenderedPath(),
	}
}

func (g *Generator) PagesToTemplateContents(indexPage *content.WebPage) [][]TemplateContent {
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
