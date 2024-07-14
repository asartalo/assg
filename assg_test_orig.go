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
