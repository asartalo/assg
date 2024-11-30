package generator

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/asartalo/assg/internal/content"
)

type ContentNode struct {
	Page   *content.WebPage
	Parent string
}

type TermMap map[string][]*content.WebPage

type ContentHierarchy struct {
	Pages         map[string]*ContentNode
	Taxonomies    map[string]TermMap
	TaxonomyPage  map[string]*content.WebPage
	childrenCache map[string][]*content.WebPage
	StaticFiles   map[string]string
	verbose       bool
}

func NewPageHierarchy(verbose bool) *ContentHierarchy {
	return &ContentHierarchy{
		Pages:        make(map[string]*ContentNode),
		TaxonomyPage: make(map[string]*content.WebPage),
		StaticFiles:  make(map[string]string),
		verbose:      verbose,
	}
}

func (ph *ContentHierarchy) Println(args ...interface{}) {
	if ph.verbose {
		fmt.Println(args...)
	}
}

func (ph *ContentHierarchy) Printf(format string, args ...interface{}) {
	if ph.verbose {
		fmt.Printf(format, args...)
	}
}

func (ph *ContentHierarchy) AddPage(page *content.WebPage) {
	ph.Println("Adding page:", page.RenderedPath())
	taxonomies := page.FrontMatter.Taxonomies
	if ph.Taxonomies == nil {
		ph.Taxonomies = make(map[string]TermMap)
	}

	for taxonomy, terms := range taxonomies {
		for _, term := range terms {
			if ph.Taxonomies[taxonomy] == nil {
				ph.Taxonomies[taxonomy] = make(TermMap)
			}
			ph.Taxonomies[taxonomy][term] = append(ph.Taxonomies[taxonomy][term], page)
		}
	}
	ph.Pages[page.RenderedPath()] = &ContentNode{
		Page: page,
	}

	if page.IsTaxonomy() {
		ph.Printf("  It's a taxonomy page: %s\n", page.TaxonomyType())
		ph.TaxonomyPage[page.TaxonomyType()] = page
	}
}

func (ph *ContentHierarchy) AddStaticFile(relPath, fullPath string) {
	ph.Println("Adding static file:", relPath)
	ph.StaticFiles[relPath] = fullPath
}

func (ph *ContentHierarchy) SortedPages() []*content.WebPage {
	pages := make([]*content.WebPage, 0, len(ph.Pages))
	for _, node := range ph.Pages {
		pages = append(pages, node.Page)
	}

	slices.SortStableFunc(pages, comparePageByDate)
	return pages
}

func (ph *ContentHierarchy) GetTaxonomyTerms(taxonomy string) TermMap {
	return ph.Taxonomies[taxonomy]
}

func (ph *ContentHierarchy) GetTaxonomyPage(taxonomy string) *content.WebPage {
	return ph.TaxonomyPage[taxonomy]
}

var comparePageByDate = func(a, b *content.WebPage) int {
	return cmp.Compare(b.DateUnixEpoch(), a.DateUnixEpoch())
}

func (ph *ContentHierarchy) Retree() {
	ph.Println("Recalculating hierarchies...")

	for path, node := range ph.Pages {
		ph.Println("  Checking:", path)
		possibleParent := filepath.Dir(path)
		parent := ""
		_, ok := ph.Pages[possibleParent]
		if ok {
			ph.Println("  Parent found:", possibleParent)
			parent = possibleParent
		} else {
			ph.Println("  Not a child")
		}

		node.Parent = parent
	}
	for taxonomy, termMap := range ph.Taxonomies {
		for term, pages := range termMap {
			slices.SortStableFunc(pages, comparePageByDate)
			ph.Taxonomies[taxonomy][term] = pages
		}
	}
}

func (ph *ContentHierarchy) GetChildren(page content.WebPage) []*content.WebPage {
	path := page.RenderedPath()
	if ph.childrenCache == nil {
		ph.childrenCache = make(map[string][]*content.WebPage)
	}

	if ph.childrenCache[path] != nil {
		return ph.childrenCache[path]
	}

	children := []*content.WebPage{}
	for _, node := range ph.Pages {
		if node.Parent == path {
			children = append(children, node.Page)
		}
	}

	slices.SortStableFunc(children, comparePageByDate)
	ph.childrenCache[path] = children

	return children
}

func (ph *ContentHierarchy) GetPage(path string) *content.WebPage {
	node, ok := ph.Pages[path]
	if ok {
		return node.Page
	}

	return nil
}

func (ph *ContentHierarchy) GetParent(page content.WebPage) *content.WebPage {
	path := page.RenderedPath()
	node, ok := ph.Pages[path]
	if ok && node.Parent != "" {
		return ph.Pages[node.Parent].Page
	}

	return nil
}

func (ph *ContentHierarchy) GetNextPage(parent *content.WebPage, child *content.WebPage) *content.WebPage {
	children := ph.GetChildren(*parent)
	for i, page := range children {
		if page.RenderedPath() == child.RenderedPath() {
			if i+1 < len(children) {
				return children[i+1]
			}
		}
	}

	return nil
}

func (ph *ContentHierarchy) GetPrevPage(parent *content.WebPage, child *content.WebPage) *content.WebPage {
	children := ph.GetChildren(*parent)
	for i, page := range children {
		if page.RenderedPath() == child.RenderedPath() {
			if i-1 >= 0 {
				return children[i-1]
			}
		}
	}

	return nil
}
