package generator

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

// ... (FrontMatter and Page structs remain the same)

// IndexFrontMatter represents the TOML frontmatter of an index file.
type IndexFrontMatter struct {
	Title        string `yaml:"title"`
	SortBy       string `yaml:"sort_by"`
	Template     string `yaml:"template"`
	PageTemplate string `yaml:"page_template"`
	PaginateBy   int    `yaml:"paginate_by"`
}

// Index represents an index file (e.g., `_index.md` files).
type Index struct {
	FrontMatter IndexFrontMatter
	Markdown    string
}

// ParseIndex parses an index file with its specific frontmatter.
func ParseIndex(content []byte) (*Index, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := MdParser.Convert(content, &buf, parser.WithContext(context)); err != nil {
		return nil, err
	}

	// Extract frontmatter from the context
	fm := IndexFrontMatter{}
	if fmi := frontmatter.Get(context); fmi != nil {
		if err := fmi.Decode(&fm); err != nil {
			return nil, fmt.Errorf("failed to decode index front matter: %v", err)
		}
	}

	// Extract Markdown from the buffer
	markdown := buf.String()

	return &Index{FrontMatter: fm, Markdown: markdown}, nil
}
