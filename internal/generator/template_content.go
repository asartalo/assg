package generator

import (
	htmltpl "html/template"

	"github.com/asartalo/assg/internal/config"
)

type TemplateContent struct {
	FrontMatter
	Content   htmltpl.HTML
	Config    config.Config
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
