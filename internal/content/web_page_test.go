package content

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParsingPageWithNoIndex(t *testing.T) {
	a := assert.New(t)
	md := `+++
title = "Test Page"
date = "2024-02-01T10:00:00Z"
description = "Test page description"
+++

Hello.
`
	page, err := ParsePage("test.md", []byte(md))

	a.NoError(err)
	a.Equal("Test Page", page.FrontMatter.Title)
	a.Equal(time.Date(2024, time.February, 1, 10, 0, 0, 0, time.UTC), page.FrontMatter.Date)
	a.Equal("Test page description", page.FrontMatter.Description)
	a.Equal("<p>Hello.</p>\n", page.Content.String())
	a.Equal(false, page.FrontMatter.Draft)
	a.Equal(false, page.IsIndex())
}

func TestParsingPageWithExtraData(t *testing.T) {
	a := assert.New(t)
	md := `+++
title = "Extra Test Page"
date = "2024-02-01T10:00:00Z"
description = "Test page with extra data"

[extra]
key = "value"
number = 1
+++

Extra extra.
`
	page, err := ParsePage("test.md", []byte(md))

	a.NoError(err)
	a.Equal("Extra Test Page", page.FrontMatter.Title)
	a.Equal(time.Date(2024, time.February, 1, 10, 0, 0, 0, time.UTC), page.FrontMatter.Date)
	a.Equal("Test page with extra data", page.FrontMatter.Description)
	a.Equal("<p>Extra extra.</p>\n", page.Content.String())
	a.Equal(true, page.FrontMatter.HasExtraData("key"))
	a.Equal(false, page.FrontMatter.HasExtraData("foo"))
	a.Equal("value", page.FrontMatter.GetExtraData("key"))
	a.Equal(int64(1), page.FrontMatter.GetExtraData("number"))
}

func TestParsingPageWithIndex(t *testing.T) {
	a := assert.New(t)
	md := `+++
title = "Test Index Page"
date = "2024-02-02T10:00:00Z"
description = "Test index page description"

[index]
template = "page_index.html"
page_template = "page.html"
sort_by = "date"
paginate_by = 10
+++

Index page content.
`
	page, err := ParsePage("index.md", []byte(md))

	a.NoError(err)
	a.Equal("Test Index Page", page.FrontMatter.Title)
	a.Equal(time.Date(2024, time.February, 2, 10, 0, 0, 0, time.UTC), page.FrontMatter.Date)
	a.Equal("Test index page description", page.FrontMatter.Description)
	a.Equal("<p>Index page content.</p>\n", page.Content.String())

	a.Equal("page_index.html", page.FrontMatter.Index.Template)
	a.Equal("page.html", page.FrontMatter.Index.PageTemplate)
	a.Equal("date", page.FrontMatter.Index.SortBy)
	a.Equal(10, page.FrontMatter.Index.PaginateBy)
}
