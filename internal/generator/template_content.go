package generator

import (
	"cmp"
	htmltpl "html/template"
	"slices"
)

type TemplateContent struct {
	FrontMatter
	Content htmltpl.HTML
}

type IndexTemplateContent struct {
	TemplateContent
	Pages []TemplateContent
	Prev  string
	Next  string
}

func PageToTemplateContent(page *WebPage) TemplateContent {
	return TemplateContent{
		FrontMatter: page.FrontMatter,
		Content:     htmltpl.HTML(string(page.Content.String())),
	}
}

func PagesToTemplateContents(indexPage *WebPage, hierarchy ContentHierarchy) [][]TemplateContent {
	childPages := hierarchy.GetChildren(*indexPage)
	// sort by  date
	slices.SortStableFunc(childPages, func(a, b *WebPage) int {
		return cmp.Compare(b.DateUnixEpoch(), a.DateUnixEpoch())
	})

	paginateBy := indexPage.FrontMatter.Index.PaginateBy
	return PaginateTransform(childPages, paginateBy, PageToTemplateContent)
}
