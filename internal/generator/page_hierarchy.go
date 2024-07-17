package generator

import "path/filepath"

type PageNode struct {
	Page   *Page
	Parent string
}

type PageHierarchy struct {
	Pages map[string]*PageNode
}

func NewPageHierarchy() *PageHierarchy {
	return &PageHierarchy{
		Pages: make(map[string]*PageNode),
	}
}

// TODO: REDO so that we can get parent without rewalking tree? Or maybe we should...
func (ph *PageHierarchy) AddPage(page *Page) {
	ph.Pages[page.RenderedPath()] = &PageNode{
		Page: page,
	}
}

func (ph *PageHierarchy) Retree() {
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

func (ph *PageHierarchy) GetChildren(page Page) []*Page {
	path := page.RenderedPath()
	children := []*Page{}
	for _, node := range ph.Pages {
		if node.Parent == path {
			children = append(children, node.Page)
		}
	}

	return children
}

func (ph *PageHierarchy) GetParent(page Page) *Page {
	path := page.RenderedPath()
	node, ok := ph.Pages[path]
	if ok && node.Parent != "" {
		return ph.Pages[node.Parent].Page
	}

	return nil
}
