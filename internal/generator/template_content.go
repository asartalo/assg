package generator

import (
	htmltpl "html/template"
)

type TemplateContent struct {
	FrontMatter
	Content htmltpl.HTML
}

type PaginatedTemplateContent struct {
	TemplateContent
	Prev     string
	PrevPage TemplateContent
	Next     string
	NextPage TemplateContent
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

	paginateBy := indexPage.FrontMatter.Index.PaginateBy
	return PaginateTransform(childPages, paginateBy, PageToTemplateContent)
}
