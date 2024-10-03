package generator

import (
	htmltpl "html/template"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/content"
)

type TemplateContent struct {
	content.FrontMatter
	Content   htmltpl.HTML
	Config    config.Config
	RootPath  string
	Permalink string
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
