package generator

import (
	htmltpl "html/template"

	"codeberg.org/asartalo/assg/internal/config"
	"codeberg.org/asartalo/assg/internal/content"
)

type TemplateContent struct {
	content.FrontMatter
	Content   htmltpl.HTML
	Config    config.Config
	Path      string
	RootPath  string
	Permalink string
	Summary   htmltpl.HTML
}

func (t TemplateContent) HasExtra(key string) bool {
	_, ok := t.Extra[key]
	return ok
}

func (t TemplateContent) GetExtra(key string) any {
	return t.Extra[key]
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
	Pages       []TemplateContent
	Prev        string
	Next        string
	CurrentPage int
	TotalPages  int
}

type TaxonomyTermContent struct {
	Term      string
	PageCount int
	Permalink string
	RootPath  string
}

type TermIndexTemplateContent struct {
	TaxonomyTermContent
	IndexTemplateContent
}
