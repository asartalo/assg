package template

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	mset "github.com/deckarep/golang-set/v2"
)

type Engine struct {
	Templates map[string]*template.Template
	funcMap   template.FuncMap
}

func New(funcMap template.FuncMap) *Engine {
	return &Engine{funcMap: funcs(funcMap)}
}

func FirstParagraphFromHtml(htmlContent template.HTML) template.HTML {
	// Regular expression to find the first paragraph
	paragraphRegex := regexp.MustCompile(`(?s)<p>(.+?)</p>`)

	// Find the first paragraph match
	match := paragraphRegex.FindStringSubmatch(string(htmlContent))

	if len(match) > 1 {
		// Paragraph found, return it
		return template.HTML(match[0])
	}

	// No paragraph found, extract first 30 words
	words := strings.Fields(strings.TrimSpace(string(htmlContent)))
	maxWords := 30

	if len(words) > maxWords {
		words = words[:maxWords]
	}

	// Add ellipsis if needed
	if len(words) == maxWords {
		words = append(words, "...")
	}

	// Construct and return paragraph-like content
	return template.HTML("<p>" + strings.Join(words, " ") + "</p>")
}

func FirstParagraphFromString(content string) string {
	return string(FirstParagraphFromHtml(template.HTML(content)))
}

func funcs(otherFuncMap template.FuncMap) template.FuncMap {
	initMap := sprig.HtmlFuncMap()
	initMap["firstParagraph"] = FirstParagraphFromHtml
	initMap["timeAttr"] = func(time time.Time) string {
		return time.Format("2006-01-02T15:04:05Z07:00")
	}

	for k, v := range otherFuncMap {
		initMap[k] = v
	}

	return initMap
}

type templateInfo struct {
	Contents   string
	references mset.Set[string]
}

func getReferences(contents []byte) mset.Set[string] {
	refs := mset.NewSet[string]()
	// Regular expression to find template expression {{ template "foo.html" . }}
	// This regex will match the template expression and capture the template name
	// in the first capture group
	templateRegex := regexp.MustCompile(`{{\s*template\s+"([^"]+)"\s*\.\s*}}`)
	result := templateRegex.FindAllSubmatch(contents, -1)
	for _, match := range result {
		refs.Add(string(match[1]))
	}

	return refs
}

func (e *Engine) LoadTemplates(templateDir string) error {
	fileInfos := make(map[string]templateInfo)
	e.Templates = make(map[string]*template.Template)

	err := filepath.WalkDir(templateDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".html" {
			relPath, err := filepath.Rel(templateDir, path)
			if err != nil {
				return err
			}

			// tmpTemplate := e.assignTemplate(relPath)

			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			fileInfos[relPath] = templateInfo{
				Contents:   string(contents),
				references: getReferences(contents),
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	for name, info := range fileInfos {
		tmpTemplate := template.New(name)
		tmpTemplate = tmpTemplate.Funcs(e.funcMap)
		_, err = tmpTemplate.Parse(info.Contents)
		if err != nil {
			return err
		}

		referenceTracker := mset.NewSet[string]()
		referenceTracker.Add(name)
		for _, ref := range info.references.ToSlice() {
			if referenceTracker.Contains(ref) {
				continue
			}

			refInfo, ok := fileInfos[ref]
			if !ok {
				continue
			}

			atmpl := tmpTemplate.New(ref)
			atmpl.Parse(refInfo.Contents)
		}

		e.Templates[name] = tmpTemplate
	}

	redirectTmpl := template.New("_redirect")
	redirectTmpl.Parse(redirectTemplate)
	e.Templates["_redirect"] = redirectTmpl

	return err
}

func (e *Engine) RenderTemplate(name string, result io.Writer, data interface{}) error {
	tmpl, ok := e.Templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	return tmpl.ExecuteTemplate(result, name, data)
}

func (e *Engine) TemplateExists(name string) bool {
	_, ok := e.Templates[name]
	return ok
}

var redirectTemplate = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<link rel="canonical" href="{{ . }}">
	<meta http-equiv="refresh" content="0; url={{ . }}">
	<title>Redirect</title>
</head>
<body>
	<p><a href="{{ . }}">Click here</a> to be redirected.</p>
</body>
</html>
`
