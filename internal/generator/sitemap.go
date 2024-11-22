package generator

import (
	"encoding/xml"
	"io"
)

// Sitemap represents the sitemap of the site.
type Sitemap struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	Urls    []*SitemapUrl
}

// SitemapUrl represents a URL in the sitemap.
type SitemapUrl struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
}

// WriteXML writes the sitemap to the writer.
func (s *Sitemap) WriteXML(wr io.Writer) error {
	output, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	_, err = wr.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	_, err = wr.Write(output)
	if err != nil {
		return err
	}

	_, err = wr.Write([]byte("\n"))

	return err
}
