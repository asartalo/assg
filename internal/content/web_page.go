package content

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/asartalo/assg/internal/markdown"
	"github.com/asartalo/assg/internal/template"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

type IndexFields struct {
	SortBy       string `toml:"sort_by"`
	Template     string `toml:"template"`
	PageTemplate string `toml:"page_template"`
	PaginateBy   int    `toml:"paginate_by"`
	Taxonomy     string `toml:"taxonomy"`
}

func firstNonEmptyString(strs ...string) string {
	for _, str := range strs {
		str = strings.TrimSpace(str)
		if str != "" {
			return str
		}
	}

	return ""
}

// FrontMatter represents the TOML frontmatter of a Markdown file.
type FrontMatter struct {
	Title       string              `toml:"title"`
	Description string              `toml:"description"`
	Date        time.Time           `toml:"date"`
	Draft       bool                `toml:"draft"`
	Summary     string              `toml:"summary"`
	Taxonomies  map[string][]string `toml:"taxonomies"`
	Template    string              `toml:"template"`
	Index       IndexFields         `toml:"index"`
	Extra       map[string]any      `toml:"extra"`
}

func (f FrontMatter) HasExtraData(key string) bool {
	_, ok := f.Extra[key]
	return ok
}

func (f FrontMatter) GetExtraData(key string) interface{} {
	return f.Extra[key]
}

// WebPage represents the parsed content of a Markdown file.
type WebPage struct {
	FrontMatter    FrontMatter
	Content        bytes.Buffer
	MarkdownPath   string
	contentSummary string
}

func (p *WebPage) DateUnixEpoch() int64 {
	return p.FrontMatter.Date.UnixMilli()
}

// IsIndex returns true if the page is an index page.
func (p *WebPage) IsIndex() bool {
	return p.FrontMatter.Index.SortBy != ""
}

func (p *WebPage) IsTaxonomy() bool {
	return p.IsIndex() && p.TaxonomyType() != ""
}

func (p *WebPage) TaxonomyType() string {
	return p.FrontMatter.Index.Taxonomy
}

func RootPath(path string) string {
	path = fmt.Sprintf("/%s/", path)
	if path == "//" {
		return "/"
	}

	return path
}

func (p *WebPage) RootPath() string {
	return RootPath(filepath.ToSlash(p.RenderedPath()))
}

func (p *WebPage) Summary() (string, error) {
	if p.contentSummary != "" {
		return p.contentSummary, nil
	}

	summaryAvailable := firstNonEmptyString(p.FrontMatter.Summary, p.FrontMatter.Description)
	rendered := bytes.Buffer{}
	if summaryAvailable != "" {
		context := parser.NewContext()
		if err := markdown.Parser.Convert([]byte(summaryAvailable), &rendered, parser.WithContext(context)); err != nil {
			return "", err
		}

		p.contentSummary = strings.TrimSpace(rendered.String())
	} else {
		p.contentSummary = template.FirstParagraphFromString(p.Content.String())
	}

	return p.contentSummary, nil
}

// ParsePage parses a Markdown file with TOML frontmatter.
func ParsePage(path string, content []byte) (*WebPage, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := markdown.Parser.Convert(content, &buf, parser.WithContext(context)); err != nil {
		return nil, err
	}

	// Extract frontmatter from the context
	fm := FrontMatter{}
	if fmi := frontmatter.Get(context); fmi != nil {
		if err := fmi.Decode(&fm); err != nil {
			return nil, fmt.Errorf("failed to decode front matter: %v", err)
		}
	}

	return &WebPage{FrontMatter: fm, Content: buf, MarkdownPath: path}, nil
}

func (p *WebPage) RenderedPath() string {
	// if the file is named index.md, we want to render it as the root index.html (e.g. /index.html)
	if p.MarkdownPath == "index.md" {
		return ""
	}

	extension := filepath.Ext(p.MarkdownPath)
	lastDotIndex := len(p.MarkdownPath) - len(extension)
	return p.MarkdownPath[:lastDotIndex]
}

func (p *WebPage) IsDraft() bool {
	return p.FrontMatter.Draft
}
