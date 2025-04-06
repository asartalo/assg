package generator

import (
	"cmp"
	"fmt"
	htmltpl "html/template"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/content"
	"github.com/asartalo/assg/internal/template"
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
	ag            *AtomGenerator
	pg            *PageGenerator
}

func defineFuncs(generator *Generator) htmltpl.FuncMap {
	funcMap := make(htmltpl.FuncMap)
	funcMap["sectionPages"] = func(indexPath string, max, offset int) []TemplateContent {
		return generator.pg.GetSectionPages(indexPath, max, offset)
	}

	funcMap["sectionIndex"] = func(indexPath string) IndexTemplateContent {
		page := generator.hierarchy.GetPage(indexPath)
		if page == nil {
			fmt.Printf("Unable to find page \"%s\"\n", indexPath)
		}
		templateContent := generator.pg.PageToTemplateContent(page)

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
		return htmltpl.HTML(generator.ag.AtomLinks())
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

	generator.hierarchy = NewPageHierarchy(ContentHierarchyOptions{
		IncludeDrafts: cfg.IncludeDrafts,
		Verbose:       verbose,
	})

	funcMap := defineFuncs(generator)
	templates := template.New(funcMap)
	err = templates.LoadTemplates(path.Join(srcDir, "templates"))
	if err != nil {
		return nil, err
	}

	generator.Tmpl = templates
	generator.taxonomyCache = make(map[string]TermTTC)

	generator.ag = &AtomGenerator{
		mg:     generator,
		Config: generator.Config,
	}
	generator.pg = &PageGenerator{
		mg:        generator,
		Config:    cfg,
		hierarchy: generator.hierarchy,
	}

	return generator, err
}

func (g *Generator) Println(args ...any) {
	if g.verbose {
		fmt.Println(args...)
	}
}

func (g *Generator) Printf(format string, args ...any) {
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
		err := g.pg.GeneratePage(node.Page, now)
		if err != nil {
			return err
		}
	}

	g.Println("\nCopying static files...")
	err = g.CopyStaticFiles()
	if err != nil {
		return err
	}

	err = g.ag.GenerateFeed(now)
	if err != nil {
		fmt.Println("IT ERRORd!")
		fmt.Println(err)
		return err
	}

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

	err = sitemap.WriteXML(sitemapFile)
	if err != nil {
		return err
	}

	return sitemapFile.Close()
}

const DEFAULT_TEMPLATE = "default.html"

func (g *Generator) OutputPath(endPath string) string {
	return path.Join(g.Config.OutputDirectoryAbsolute(), endPath)
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

func (g *Generator) pathValue(templateData any) string {
	switch v := templateData.(type) {
	case string:
		return v
	case TemplateContent:
		return v.Path
	default:
		return ""
	}
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
	// open source file for reading
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
