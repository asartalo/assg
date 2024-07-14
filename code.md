./go.mod

```mod
module github.com/asartalo/assg

go 1.22.1

require (
	github.com/mangoumbrella/goldmark-figure v1.2.0
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/yuin/goldmark v1.7.4 // indirect
	go.abhg.dev/goldmark/frontmatter v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

./internal/template/template.go

```go
package template

import (
	"html/template"
)

type Engine struct {
	templates *template.Template
}

func New() *Engine {
	return &Engine{
		templates: template.New("").Funcs(templateFuncs()),
	}
}

func (e *Engine) LoadTemplates(pattern string) error {
	var err error
	e.templates, err = e.templates.ParseGlob(pattern)
	return err
}

func (e *Engine) Render(name string, data interface{}) (string, error) {
	// Implement rendering logic
	return "", nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// Define custom template functions
	}
}
```

./internal/config/config.go

```go
package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	BaseURL         string                 `toml:"base_url"`
	Title           string                 `toml:"title"`
	Description     string                 `toml:"description"`
	DefaultLanguage string                 `toml:"default_language"`
	Author          string                 `toml:"author"`
	CompileSass     bool                   `toml:"compile_sass"`
	GenerateFeed    bool                   `toml:"generate_feed"`
	FeedLimit       int                    `toml:"feed_limit"`
	Taxonomies      []TaxonomyConfig       `toml:"taxonomies"`
	Markdown        MarkdownConfig         `toml:"markdown"`
	Extra           map[string]interface{} `toml:"extra"`
}

type TaxonomyConfig struct {
	Name       string `toml:"name"`
	Feed       bool   `toml:"feed"`
	PaginateBy int    `toml:"paginate_by"` // Add this field for optional pagination
}

type MarkdownConfig struct {
	HighlightCode    bool `toml:"highlight_code"`
	SmartPunctuation bool `toml:"smart_punctuation"`
}

func Load(filename string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
```

./internal/generator/generator.go

```go
package generator

import (
	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/template"
)

type Generator struct {
	Config *config.Config
	Tmpl   *template.Engine
}

func New(cfg *config.Config) *Generator {
	return &Generator{
		Config: cfg,
		Tmpl:   template.New(),
	}
}

func (g *Generator) Generate(outputDir string) error {
	return nil
}
```

./internal/generator/md_parser.go

```go
package generator

import (
	figure "github.com/mangoumbrella/goldmark-figure"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
)

var MdParser = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Footnote,
		&frontmatter.Extender{},
		figure.Figure,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
	),
)
```

./internal/generator/index.go

```go
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
```

./internal/generator/page.go

```go
package generator

import (
	"bytes"
	"fmt"
	"time"

	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

type IndexFields struct {
	SortBy       string `yaml:"sort_by"`
	Template     string `yaml:"template"`
	PageTemplate string `yaml:"page_template"`
	PaginateBy   int    `yaml:"paginate_by"`
}

// PageFrontMatter represents the TOML frontmatter of a Markdown file.
type PageFrontMatter struct {
	Title      string              `toml:"title"`
	Date       time.Time           `toml:"date"`
	Draft      bool                `toml:"draft"`
	Taxonomies map[string][]string `toml:"taxonomies"`
	Index      IndexFields         `toml:"index"`
}

// Page represents the parsed content of a Markdown file.
type Page struct {
	FrontMatter PageFrontMatter
	Markdown    string
	Path        string
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

	// Extract Markdown from the buffer
	markdown := buf.String()

	return &Page{FrontMatter: fm, Markdown: markdown, Path: path}, nil
}
```

./internal/commands/build.go

```go
package commands

func Build(srcDir, outputDir string, includeDrafts bool) error {
	return nil
}
```

./assg_test.go

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestASSGEndToEnd(t *testing.T) {
	// Setup test environment
	testDir, err := os.MkdirTemp("", "assg-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test site structure
	createTestSiteStructure(t, testDir)

	// Run assg build
	err = runASSGBuild(testDir)
	if err != nil {
		t.Fatalf("Failed to run assg build: %v", err)
	}

	// Verify the build output
	t.Run("VerifyBuildOutput", func(t *testing.T) {
		verifyBuildOutput(t, testDir)
	})

	// Test draft pages
	t.Run("TestDraftPages", func(t *testing.T) {
		testDraftPages(t, testDir)
	})

	// Test pagination
	t.Run("TestPagination", func(t *testing.T) {
		testPagination(t, testDir)
	})

	// Test taxonomies
	t.Run("TestTaxonomies", func(t *testing.T) {
		testTaxonomies(t, testDir)
	})

	// Test future dated posts
	t.Run("TestFutureDatedPosts", func(t *testing.T) {
		testFutureDatedPosts(t, testDir)
	})

	// Test non-markdown file mirroring
	t.Run("TestNonMarkdownMirroring", func(t *testing.T) {
		testNonMarkdownMirroring(t, testDir)
	})
}

func createTestSiteStructure(t *testing.T, testDir string) {
	// Create directories
	dirs := []string{"content", "templates", "public"}
	for _, dir := range dirs {
		err := os.Mkdir(filepath.Join(testDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create config.toml
	configContent := `
base_url = "http://example.com/"
title = "Test Blog"
description = "A test blog for ASSG"
generate_feed = true
feed_limit = 10

[markdown]
highlight_code = true
smart_punctuation = true
`
	err := os.WriteFile(filepath.Join(testDir, "config.toml"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config.toml: %v", err)
	}

	// Create content files
	createContentFiles(t, testDir)

	// Create template files
	createTemplateFiles(t, testDir)
}

func createContentFiles(t *testing.T, testDir string) {
	contentDir := filepath.Join(testDir, "content")

	// Create regular post
	regularPost := `+++
title = "Regular Post"
date = "2024-03-01T10:00:00Z"
description = "This is a regular post"
[taxonomies]
tags = ["test", "regular"]
+++
This is the content of a regular post.
`
	err := os.WriteFile(filepath.Join(contentDir, "regular-post.md"), []byte(regularPost), 0644)
	if err != nil {
		t.Fatalf("Failed to create regular post: %v", err)
	}

	// Create draft post
	draftPost := `+++
title = "Draft Post"
date = "2024-03-02T10:00:00Z"
description = "This is a draft post"
draft = true
[taxonomies]
tags = ["test", "draft"]
+++
This is the content of a draft post.
`
	err = os.WriteFile(filepath.Join(contentDir, "draft-post.md"), []byte(draftPost), 0644)
	if err != nil {
		t.Fatalf("Failed to create draft post: %v", err)
	}

	// Create future dated post
	futureDatedPost := `+++
title = "Future Post"
date = "2025-01-01T10:00:00Z"
description = "This is a future dated post"
[taxonomies]
tags = ["test", "future"]
+++
This is the content of a future dated post.
`
	err = os.WriteFile(filepath.Join(contentDir, "future-post.md"), []byte(futureDatedPost), 0644)
	if err != nil {
		t.Fatalf("Failed to create future dated post: %v", err)
	}

	// Create index page
	indexPage := `+++
title = "Blog Posts"
description = "Index of blog posts"
[index]
sort_by = "date"
template = "index.html"
page_template = "page.html"
paginate_by = 2
+++
This is the index page content.
`
	err = os.WriteFile(filepath.Join(contentDir, "_index.md"), []byte(indexPage), 0644)
	if err != nil {
		t.Fatalf("Failed to create index page: %v", err)
	}

	// Create a non-markdown file
	err = os.WriteFile(filepath.Join(contentDir, "image.jpg"), []byte("fake image content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create non-markdown file: %v", err)
	}
}

func createTemplateFiles(t *testing.T, testDir string) {
	templatesDir := filepath.Join(testDir, "templates")

	// Create base template
	baseTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>{{ .Title }}</title>
    <meta name="description" content="{{ .Description }}">
</head>
<body>
    {{ block "content" . }}{{ end }}
</body>
</html>
`
	err := os.WriteFile(filepath.Join(templatesDir, "base.html"), []byte(baseTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create base template: %v", err)
	}

	// Create index template
	indexTemplate := `
{{ define "content" }}
<h1>{{ .Title }}</h1>
{{ .Content }}
<ul>
{{ range .Pages }}
    <li><a href="{{ .Permalink }}">{{ .Title }}</a></li>
{{ end }}
</ul>
{{ end }}
`
	err = os.WriteFile(filepath.Join(templatesDir, "index.html"), []byte(indexTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create index template: %v", err)
	}

	// Create page template
	pageTemplate := `
{{ define "content" }}
<h1>{{ .Title }}</h1>
<p>{{ .Date.Format "2006-01-02" }}</p>
{{ .Content }}
{{ end }}
`
	err = os.WriteFile(filepath.Join(templatesDir, "page.html"), []byte(pageTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create page template: %v", err)
	}
}

func runASSGBuild(testDir string) error {
	// Implement the actual call to your assg build command here
	// For now, we'll just simulate it by creating some output files
	publicDir := filepath.Join(testDir, "public")

	// Create index.html
	indexContent := "<html><body><h1>Blog Posts</h1><ul><li><a href='/regular-post/'>Regular Post</a></li></ul></body></html>"
	err := os.WriteFile(filepath.Join(publicDir, "index.html"), []byte(indexContent), 0644)
	if err != nil {
		return err
	}

	// Create regular-post/index.html
	regularPostContent := "<html><body><h1>Regular Post</h1><p>2024-03-01</p><p>This is the content of a regular post.</p></body></html>"
	err = os.Mkdir(filepath.Join(publicDir, "regular-post"), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(publicDir, "regular-post", "index.html"), []byte(regularPostContent), 0644)
	if err != nil {
		return err
	}

	// Mirror non-markdown file
	err = os.WriteFile(filepath.Join(publicDir, "image.jpg"), []byte("fake image content"), 0644)
	if err != nil {
		return err
	}

	return nil
}

func verifyBuildOutput(t *testing.T, testDir string) {
	publicDir := filepath.Join(testDir, "public")

	// Check if index.html exists
	if _, err := os.Stat(filepath.Join(publicDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("index.html was not generated")
	}

	// Check if regular post was generated
	if _, err := os.Stat(filepath.Join(publicDir, "regular-post", "index.html")); os.IsNotExist(err) {
		t.Errorf("regular-post/index.html was not generated")
	}

	// Check if draft post was not generated
	if _, err := os.Stat(filepath.Join(publicDir, "draft-post", "index.html")); !os.IsNotExist(err) {
		t.Errorf("draft-post/index.html was generated but shouldn't have been")
	}

	// Check if future dated post was not generated
	if _, err := os.Stat(filepath.Join(publicDir, "future-post", "index.html")); !os.IsNotExist(err) {
		t.Errorf("future-post/index.html was generated but shouldn't have been")
	}
}

func testDraftPages(t *testing.T, testDir string) {
	publicDir := filepath.Join(testDir, "public")

	// Check that draft post is not in the public directory
	if _, err := os.Stat(filepath.Join(publicDir, "draft-post", "index.html")); !os.IsNotExist(err) {
		t.Errorf("draft-post/index.html was generated but shouldn't have been")
	}

	// TODO: Implement test for --include-draft flag in serve mode
}

func testPagination(t *testing.T, testDir string) {
	publicDir := filepath.Join(testDir, "public")

	// Check for pagination pages (this depends on how many posts you have and your pagination settings)
	if _, err := os.Stat(filepath.Join(publicDir, "page", "1", "index.html")); os.IsNotExist(err) {
		t.Errorf("Pagination page 1 was not generated")
	}
}

func testTaxonomies(t *testing.T, testDir string) {
	publicDir := filepath.Join(testDir, "public")

	// Check for taxonomy pages
	if _, err := os.Stat(filepath.Join(publicDir, "tags", "index.html")); os.IsNotExist(err) {
		t.Errorf("Tags index page was not generated")
	}

	if _, err := os.Stat(filepath.Join(publicDir, "tags", "test", "index.html")); os.IsNotExist(err) {
		t.Errorf("Tag 'test' page was not generated")
	}
}

func testFutureDatedPosts(t *testing.T, testDir string) {
	publicDir := filepath.Join(testDir, "public")

	// Check that future dated post is not in the public directory
	if _, err := os.Stat(filepath.Join(publicDir, "future-post", "index.html")); !os.IsNotExist(err) {
		t.Errorf("future-post/index.html was generated but shouldn't have been")
	}
}

func testNonMarkdownMirroring(t *testing.T, testDir string) {
	publicDir := filepath.Join(testDir, "public")

	// Check if non-markdown file was mirrored
	if _, err := os.Stat(filepath.Join(publicDir, "image.jpg")); os.IsNotExist(err) {
		t.Errorf("image.jpg was not mirrored to the public directory")
	}
}
```

./cmd/assg/main.go

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/asartalo/assg/internal/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "assg",
	Short: "ASSG is Asartalo's Static Site Generator",
	Long:  `ASSG (Asartalo's Static Site Generator) is a static site generator built with love by Asartalo.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long:  `Generates the static site based on the content and templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Building site...")
		srcDir, err := os.Getwd()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		outputDir := filepath.Join(srcDir, "public")
		err = commands.Build(srcDir, outputDir, false)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the static site",
	Long:  `Starts a local server to preview the static site.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting server...")
		// Implement your serve logic here
	},
}

var includeDrafts bool

func init() {
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(serveCmd)

	// Add flags
	serveCmd.Flags().BoolVar(&includeDrafts, "include-drafts", false, "Include draft pages when serving")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

