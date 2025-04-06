package generator

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/asartalo/assg/internal/config"
	"github.com/asartalo/assg/internal/content"
)

type FeedGenerator struct {
	XMLName xml.Name `xml:"generator"`
	Uri     string   `xml:"uri,attr"`
	Name    string   `xml:",chardata"`
}

type FeedDateTime time.Time

func (t FeedDateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	orig := time.Time(t)
	return e.EncodeElement(orig.Format(time.RFC3339), start)
}

type FeedLink struct {
	XMLName xml.Name `xml:"link"`
	Rel     string   `xml:"rel,attr"`
	Type    string   `xml:"type,attr"`
	Href    string   `xml:"href,attr"`
}

type Feed struct {
	Xmlns     string   `xml:"xmlns,attr"`
	Lang      string   `xml:"xml:lang,attr"`
	XMLName   xml.Name `xml:"feed"`
	Title     string   `xml:"title"`
	Subtitle  string   `xml:"subtitle"`
	Id        string   `xml:"id"`
	Links     []*FeedLink
	Generator *FeedGenerator
	Updated   FeedDateTime `xml:"updated"`
	Entries   []*FeedEntry
}

type FeedEntry struct {
	XMLName   xml.Name          `xml:"entry"`
	Lang      string            `xml:"xml:lang,attr"`
	Title     string            `xml:"title"`
	Id        string            `xml:"id"`
	Published FeedDateTime      `xml:"published"`
	Updated   FeedDateTime      `xml:"updated"`
	Content   *FeedContent      `xml:"content,omitempty"`
	Summary   *FeedEntrySummary `xml:"summary,omitempty"`
	Authors   []*FeedAuthor
	Links     []*FeedLink
}

type FeedContent struct {
	XMLName xml.Name `xml:"content"`
	Type    string   `xml:"type,attr"`
	Src     string   `xml:"src,attr,omitempty"`
	Content string   `xml:",chardata"`
}

type FeedEntrySummary struct {
	XMLName xml.Name `xml:"summary"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

type FeedAuthor struct {
	XMLName xml.Name `xml:"author"`
	Name    string   `xml:"name"`
	Email   string   `xml:"email,omitempty"`
	Uri     string   `xml:"uri,omitempty"`
}

var LinkEndRegexp = regexp.MustCompile(`></link>`)

func formatEmptyElements(xmlBytes []byte) []byte {
	return LinkEndRegexp.ReplaceAll(xmlBytes, []byte("/>"))
}

func (f *Feed) WriteXML(atomFile io.Writer) error {
	output, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}

	_, err = atomFile.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	_, err = atomFile.Write(formatEmptyElements(output))
	if err != nil {
		return err
	}

	_, err = atomFile.Write([]byte("\n"))

	return err
}

type AtomGenerator struct {
	mg     *Generator
	Config *config.Config
}

func (ag *AtomGenerator) defaultFeedAuthor() *FeedAuthor {
	mg := ag.mg
	if mg.feedAuthor == nil {
		mg.feedAuthor = &FeedAuthor{
			Name: mg.Config.Author,
		}
	}

	return mg.feedAuthor
}

func (ag *AtomGenerator) GenerateFeed(now time.Time) error {
	mg := ag.mg
	if !mg.Config.GenerateFeed {
		return nil
	}

	atomUrl := mg.FullUrl("atom.xml")
	feed := Feed{
		Xmlns:     "http://www.w3.org/2005/Atom",
		Lang:      "en",
		Title:     mg.Config.Title,
		Subtitle:  mg.Config.Description,
		Id:        atomUrl,
		Generator: &FeedGenerator{Uri: "https://github.com/asartalo/assg", Name: "ASSG"},
		Updated:   FeedDateTime(now),
		Links: []*FeedLink{
			{
				Rel:  "self",
				Type: "application/atom+xml",
				Href: atomUrl,
			},
			{
				Rel:  "alternate",
				Type: "text/html",
				Href: mg.SiteUrlNoTrailingslash(),
			},
		},
	}

	for _, page := range mg.hierarchy.SortedPages() {
		if page.IsTaxonomy() || page.IsIndex() {
			continue
		}

		entry, err := ag.createFeedEntry(page)
		if err != nil {
			return err
		}

		feed.Entries = append(feed.Entries, entry)
	}

	atomFilePath := mg.OutputPath("atom.xml")

	atomFile, err := os.OpenFile(atomFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	err = feed.WriteXML(atomFile)
	if err != nil {
		return err
	}

	return atomFile.Close()
}

func (ag *AtomGenerator) createFeedEntry(page *content.WebPage) (*FeedEntry, error) {
	g := ag.mg
	pageUrl := g.FullUrl(page.RootPath())

	item := &FeedEntry{
		Lang:  "en",
		Title: page.FrontMatter.Title,
		Links: []*FeedLink{
			{Rel: "alternate", Type: "text/html", Href: pageUrl},
		},
		Published: FeedDateTime(page.FrontMatter.Date),
		Updated:   FeedDateTime(page.FrontMatter.Date),
		Id:        pageUrl,
		Authors:   []*FeedAuthor{ag.defaultFeedAuthor()},
	}

	contentLength := page.Content.Len()
	// If the content is too long, or empty, use the summary
	if contentLength > 500 || contentLength == 0 {
		summary, err := page.Summary()
		if err != nil {
			return nil, err
		}

		item.Summary = &FeedEntrySummary{
			Type:    "html",
			Content: summary,
		}
		// If the content is too short, use that instead
	} else {
		item.Content = &FeedContent{
			Type:    "html",
			Content: strings.TrimSpace(page.Content.String()),
		}
	}

	return item, nil
}

func (ag *AtomGenerator) AtomLinks() string {
	feed := ag.Config.FeedsForContent[0]
	return fmt.Sprintf(
		`<link rel="alternate" title="%s" type="application/atom+xml" href="%s">`,
		ag.feedTitle(feed),
		ag.mg.FullUrl("atom.xml"),
	)
}

func (ag *AtomGenerator) feedTitle(feed config.ContentFeed) string {
	if feed.Title == "" {
		return fmt.Sprintf("%s Feed", ag.Config.Title)
	}

	return feed.Title
}
