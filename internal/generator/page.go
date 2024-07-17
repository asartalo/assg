package generator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

type IndexFields struct {
	SortBy       string `toml:"sort_by"`
	Template     string `toml:"template"`
	PageTemplate string `toml:"page_template"`
	PaginateBy   int    `toml:"paginate_by"`
}

// PageFrontMatter represents the TOML frontmatter of a Markdown file.
type PageFrontMatter struct {
	Title       string              `toml:"title"`
	Description string              `toml:"description"`
	Date        time.Time           `toml:"date"`
	Draft       bool                `toml:"draft"`
	Taxonomies  map[string][]string `toml:"taxonomies"`
	Template    string              `toml:"template"`
	Index       IndexFields         `toml:"index"`
}

// Page represents the parsed content of a Markdown file.
type Page struct {
	FrontMatter PageFrontMatter
	Content     bytes.Buffer
	Path        string
}

// IsIndex returns true if the page is an index page.
func (p *Page) IsIndex() bool {
	return p.FrontMatter.Index.SortBy != ""
}

// ParsePage parses a Markdown file with TOML frontmatter.
func ParsePage(path string, content []byte) (*Page, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := MdParser.Convert(content, &buf, parser.WithContext(context)); err != nil {
		return nil, err
	}

	// Extract frontmatter from the context
	fm := PageFrontMatter{}
	if fmi := frontmatter.Get(context); fmi != nil {
		if err := fmi.Decode(&fm); err != nil {
			return nil, fmt.Errorf("failed to decode front matter: %v", err)
		}
	}

	return &Page{FrontMatter: fm, Content: buf, Path: path}, nil
}

func (p *Page) RenderedPath() string {
	// if the file is named index.md, we want to render it as the root index.html (e.g. /index.html)
	if p.Path == "index.md" {
		return ""
	}

	extension := filepath.Ext(p.Path)
	lastDotIndex := len(p.Path) - len(extension)
	return p.Path[:lastDotIndex]
}
