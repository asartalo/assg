package generator

import (
	"bytes"
	"cmp"
	"fmt"
	htmltpl "html/template"
	"io/fs"
	"os"
	"os/exec"
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

type TermTTC map[string]*TaxonomyTermContent

type Generator struct {
	Config        *config.Config
	Tmpl          *template.Engine
	hierarchy     *ContentHierarchy
	feedAuthor    *FeedAuthor
	taxonomyCache map[string]TermTTC
	verbose       bool
	renderedPaths []string
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

	funcMap["taxonomyTerms"] = func(taxonomy string) []*TaxonomyTermContent {
		return generator.GetAllTaxonomyTerms(taxonomy)
	}

	funcMap["pageTaxonomy"] = func(path, taxonomy string) []*TaxonomyTermContent {
		return generator.GetTaxonomyTermsForPage(path, taxonomy)
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

	funcMap["devScripts"] = func() htmltpl.HTML {
		if generator.Config.DevMode {
			return htmltpl.HTML(`
				<script src="http://localhost:35729/livereload.js"></script>
			`)
		}

		return htmltpl.HTML("")
	}

	return funcMap
}

func New(cfg *config.Config, verbose bool) (*Generator, error) {
	srcDir := cfg.RootDirectory()

	generator := &Generator{
		Config:  cfg,
		verbose: verbose,
	}

	err := generator.ClearOutputDirectory()
	if err != nil {
		return nil, err
	}

	generator.hierarchy = NewPageHierarchy(verbose)

	funcMap := defineFuncs(generator)
	templates := template.New(funcMap)
	err = templates.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return nil, err
	}

	generator.Tmpl = templates
	generator.taxonomyCache = make(map[string]TermTTC)

	return generator, err
}

func (g *Generator) Println(args ...interface{}) {
	if g.verbose {
		fmt.Println(args...)
	}
}

func (g *Generator) Printf(format string, args ...interface{}) {
	if g.verbose {
		fmt.Printf(format, args...)
	}
}

func (g *Generator) ClearOutputDirectory() error {
	dir, err := os.ReadDir(g.Config.OutputDirectoryAbsolute())
	if err != nil {
		return err
	}
	for _, d := range dir {
		err = os.RemoveAll(path.Join(g.Config.OutputDirectoryAbsolute(), d.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) shouldRunPreBuild() bool {
	return g.Config.PreBuildCmd != "" && !g.Config.DevMode
}

func (g *Generator) runPreBuild() error {
	g.Println("Running pre-build command...")
	return g.runBuildCommand(g.Config.PreBuildCmd)
}

func commandAndArgs(cmd string) (string, []string) {
	parts := strings.Split(cmd, " ")
	return parts[0], parts[1:]
}

func (g *Generator) runBuildCommand(cmd string) error {
	command, args := commandAndArgs(cmd)
	cm := exec.Command(command, args...)
	cm.Env = os.Environ()
	cm.Dir = g.Config.RootDirectory()
	g.Printf("Running command: %s %v\n", command, args)
	cm.Env = append(cm.Env, fmt.Sprintf("ASSG_ROOT=%s", g.Config.RootDirectory()))
	cm.Env = append(cm.Env, fmt.Sprintf("ASSG_CONTENT=%s", g.Config.ContentDirectoryAbsolute()))
	cm.Env = append(cm.Env, fmt.Sprintf("ASSG_OUTPUT=%s", g.Config.OutputDirectoryAbsolute()))
	cm.Stdout = os.Stdout
	cm.Stderr = os.Stderr

	return cm.Run()
}

func (g *Generator) Build(now time.Time) error {
	if g.shouldRunPreBuild() {
		err := g.runPreBuild()
		if err != nil {
			return err
		}
	}

	err := g.hierarchy.GatherContent(g.Config.ContentDirectoryAbsolute())
	if err != nil {
		return err
	}

	g.Println("\nBuilding site...")
	for _, node := range g.hierarchy.Pages {
		err := g.GeneratePage(node.Page)
		if err != nil {
			return err
		}
	}

	g.Println("\nCopying static files...")
	err = g.CopyStaticFiles()
	if err != nil {
		return err
	}

	g.GenerateFeed(now)

	if g.Config.Sitemap {
		err := g.GenerateSitemap()
		if err != nil {
			return err
		}
	}

	if g.shouldRunPostBuild() {
		err := g.runPostBuild()
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) CopyStaticFiles() error {
	for relPath, fullPath := range g.hierarchy.StaticFiles {
		destinationPath := g.OutputPath(relPath)
		err := os.MkdirAll(filepath.Dir(destinationPath), 0755)
		if err != nil {
			return err
		}

		g.Printf("  Copying %s to %s\n", fullPath, destinationPath)
		err = copyFile(fullPath, destinationPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) shouldRunPostBuild() bool {
	return g.Config.PostBuildCmd != "" && !g.Config.DevMode
}

func (g *Generator) runPostBuild() error {
	g.Println("Running post-build command...")
	return g.runBuildCommand(g.Config.PostBuildCmd)
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

	defer atomFile.Close()

	err = feed.WriteXML(atomFile)
	if err != nil {
		return err
	}

	return err
}

var compareAlpha = func(a, b string) int {
	return cmp.Compare(a, b)
}

func (g *Generator) GenerateSitemap() error {
	sitemap := Sitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Urls:  make([]*SitemapUrl, 0),
	}

	// Gather URLs
	slices.SortStableFunc(g.renderedPaths, compareAlpha)
	for _, path := range g.renderedPaths {
		sitemap.Urls = append(sitemap.Urls, &SitemapUrl{
			Loc: g.FullUrl(path),
		})
	}

	sitemapFilePath := g.OutputPath("sitemap.xml")

	sitemapFile, err := os.OpenFile(sitemapFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer sitemapFile.Close()

	err = sitemap.WriteXML(sitemapFile)

	return err
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

const DEFAULT_TEMPLATE = "default.html"

func (g *Generator) OutputPath(endPath string) string {
	return path.Join(g.Config.OutputDirectoryAbsolute(), endPath)
}

func (g *Generator) GeneratePage(page *content.WebPage) (err error) {
	g.Printf("Generating page: %s\n", page.MarkdownPath)
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

	g.Printf("  Using template: %s\n", templateToUse)

	pagePath := page.RenderedPath()
	templateData := g.PageToTemplateContent(page)
	g.Printf("  Destination: %s\n", pagePath)

	if page.IsTaxonomy() {
		err = g.generateTaxonomyPages(
			page,
			templateData,
			pagePath,
			templateToUse,
		)
	} else if page.IsIndex() {
		err = g.generateIndexPages(
			page,
			templateData,
			pagePath,
			templateToUse,
			g.PagesToTemplateContents(page),
		)
	} else {
		parentPage := g.hierarchy.GetParent(*page)
		if parentPage != nil {
			g.Printf("  Parent page: %s\n", parentPage.MarkdownPath)
			err = g.renderPage(
				g.generateChildPageData(page, parentPage, templateData),
				pagePath,
				templateToUse,
				true,
			)
		} else {
			g.Printf("  Not a child page\n")
			err = g.renderPage(templateData, pagePath, templateToUse, true)
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

func (g *Generator) ensurePopulatedTaxonomyCache(taxonomy string) TermTTC {
	if cached, ok := g.taxonomyCache[taxonomy]; ok {
		return cached
	}

	ttcCache := make(TermTTC)
	g.taxonomyCache[taxonomy] = ttcCache

	mapTerms := g.hierarchy.GetTaxonomyTerms(taxonomy)
	taxonomyIndexPage := g.hierarchy.GetTaxonomyPage(taxonomy)

	for term, pages := range mapTerms {
		if _, ok := ttcCache[term]; !ok {
			rootPath := content.RootPath(
				filepath.ToSlash(path.Join(taxonomyIndexPage.RenderedPath(), dashSpaces(term))),
			)
			ttc := &TaxonomyTermContent{
				Term:      term,
				PageCount: len(pages),
				RootPath:  rootPath,
				Permalink: g.FullUrl(rootPath),
			}
			ttcCache[term] = ttc
		}
	}

	g.taxonomyCache[taxonomy] = ttcCache

	return ttcCache
}

func (g *Generator) GetAllTaxonomyTerms(taxonomy string) (termTemplates []*TaxonomyTermContent) {
	ttcCache := g.ensurePopulatedTaxonomyCache(taxonomy)

	for _, cached := range ttcCache {
		termTemplates = append(termTemplates, cached)
	}

	slices.SortStableFunc(termTemplates, func(a, b *TaxonomyTermContent) int {
		return cmp.Compare(a.Term, b.Term)
	})

	return termTemplates
}

func (g *Generator) GetTaxonomyTermsForPage(rootPath string, taxonomy string) (termTemplates []*TaxonomyTermContent) {
	ofPage := g.hierarchy.GetPage(rootPath)
	ttcCache := g.ensurePopulatedTaxonomyCache(taxonomy)
	terms := ofPage.FrontMatter.Taxonomies[taxonomy]

	for _, term := range terms {
		ttc := ttcCache[term]
		termTemplates = append(termTemplates, ttc)
	}

	slices.SortStableFunc(termTemplates, func(a, b *TaxonomyTermContent) int {
		return cmp.Compare(a.Term, b.Term)
	})

	return termTemplates
}

func (g *Generator) generateTaxonomyPages(
	page *content.WebPage,
	templateData TemplateContent,
	pagePath string,
	templateToUse string,
) (err error) {
	g.Printf("  Generating taxonomy pages for: %s\n", page.MarkdownPath)
	taxonomy := page.TaxonomyType()
	termMapping := g.hierarchy.GetTaxonomyTerms(taxonomy)
	paginateBy := page.FrontMatter.Index.PaginateBy

	err = g.renderPage(templateData, pagePath, templateToUse, true)
	if err != nil {
		return err
	}

	now := time.Now()
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
	pagePath string,
	templateToUse string,
	pagingGroups [][]TemplateContent,
) (err error) {
	g.Printf("  Generating index pages for: %s\n", page.MarkdownPath)
	pagingCount := len(pagingGroups)

	// render redirect page
	if pagingCount > 1 {
		page1Path := path.Join(pagePath, "page", "1")
		redirectPath := g.FullUrl(page.RootPath())

		err = g.renderPage(redirectPath, page1Path, "_redirect", false)
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

		err = g.renderPage(indexTemplateData, destinPath, templateToUse, true)
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

func (g *Generator) pathValue(templateData interface{}) string {
	switch v := templateData.(type) {
	case string:
		return v
	case TemplateContent:
		return v.Path
	default:
		return ""
	}
}

func (g *Generator) renderPage(
	templateData interface{},
	pagePath string,
	templateToUse string,
	canonical bool,
) error {
	destinationDir := g.OutputPath(pagePath)
	err := os.MkdirAll(destinationDir, 0755)
	if err != nil {
		return err
	}

	g.Printf("  Rendering page: %s\n", g.pathValue(templateData))
	destinationPath := path.Join(destinationDir, "index.html")
	destinationFile, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		g.Printf("Error creating file")
		return err
	}
	defer destinationFile.Close()

	err = g.Tmpl.RenderTemplate(
		templateToUse,
		destinationFile,
		templateData,
	)

	if err == nil && canonical {
		g.renderedPaths = append(g.renderedPaths, content.RootPath(pagePath))
	}

	return err
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
	destination, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer destination.Close()

	// copy file
	_, err = destination.ReadFrom(source)

	return err
}

func (g *Generator) PageToTemplateContent(page *content.WebPage) TemplateContent {
	summary := ""
	actualSummary, err := page.Summary()
	if err == nil {
		summary = actualSummary
	}

	return TemplateContent{
		FrontMatter: page.FrontMatter,
		Content:     htmltpl.HTML(string(page.Content.String())),
		Config:      *g.Config,
		RootPath:    page.RootPath(),
		Permalink:   g.FullUrl(page.RootPath()),
		Path:        page.RenderedPath(),
		Summary:     htmltpl.HTML(summary),
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

func dashSpaces(str string) string {
	return strings.ReplaceAll(str, " ", "-")
}

func isMarkdown(info fs.DirEntry) bool {
	return filepath.Ext(info.Name()) == ".md"
}
