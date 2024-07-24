package generator

import "path/filepath"

type ContentNode struct {
	Page   *WebPage
	Parent string
}

type ContentHierarchy struct {
	Pages map[string]*ContentNode
}

func NewPageHierarchy() *ContentHierarchy {
	return &ContentHierarchy{
		Pages: make(map[string]*ContentNode),
	}
}

// TODO: REDO so that we can get parent without rewalking tree? Or maybe we should...
func (ph *ContentHierarchy) AddPage(page *WebPage) {
	ph.Pages[page.RenderedPath()] = &ContentNode{
		Page: page,
	}
}

func (ph *ContentHierarchy) Retree() {
	for path, node := range ph.Pages {
		possibleParent := filepath.Dir(path)
		parent := ""
		_, ok := ph.Pages[possibleParent]
		if ok {
			parent = possibleParent
		}

		node.Parent = parent
	}
}

func (ph *ContentHierarchy) GetChildren(page WebPage) []*WebPage {
	path := page.RenderedPath()
	children := []*WebPage{}
	for _, node := range ph.Pages {
		if node.Parent == path {
			children = append(children, node.Page)
		}
	}

	return children
}

func (ph *ContentHierarchy) GetParent(page WebPage) *WebPage {
	path := page.RenderedPath()
	node, ok := ph.Pages[path]
	if ok && node.Parent != "" {
		return ph.Pages[node.Parent].Page
	}

	return nil
}
