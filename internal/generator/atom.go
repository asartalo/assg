package generator

import (
	"io"
	"regexp"
	"time"

	"encoding/xml"
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
