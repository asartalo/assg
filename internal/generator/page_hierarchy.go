package generator

import (
	"cmp"
	"path/filepath"
	"slices"
)

type ContentNode struct {
	Page   *WebPage
	Parent string
}

type ContentHierarchy struct {
	Pages         map[string]*ContentNode
	childrenCache map[string][]*WebPage
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
	if ph.childrenCache == nil {
		ph.childrenCache = make(map[string][]*WebPage)
	}

	if ph.childrenCache[path] != nil {
		return ph.childrenCache[path]
	}

	children := []*WebPage{}
	for _, node := range ph.Pages {
		if node.Parent == path {
			children = append(children, node.Page)
		}
	}
	// sort by  date
	slices.SortStableFunc(children, func(a, b *WebPage) int {
		return cmp.Compare(b.DateUnixEpoch(), a.DateUnixEpoch())
	})

	ph.childrenCache[path] = children

	return children
}

func (ph *ContentHierarchy) GetPage(path string) *WebPage {
	node, ok := ph.Pages[path]
	if ok {
		return node.Page
	}

	return nil
}

func (ph *ContentHierarchy) GetParent(page WebPage) *WebPage {
	path := page.RenderedPath()
	node, ok := ph.Pages[path]
	if ok && node.Parent != "" {
		return ph.Pages[node.Parent].Page
	}

	return nil
}

func (ph *ContentHierarchy) GetNextPage(parent *WebPage, child *WebPage) *WebPage {
	children := ph.GetChildren(*parent)
	for i, page := range children {
		if page.RenderedPath() == child.RenderedPath() {
			if i+1 < len(children) {
				return children[i+1]
			}
		}
	}

	return nil
}

func (ph *ContentHierarchy) GetPrevPage(parent *WebPage, child *WebPage) *WebPage {
	children := ph.GetChildren(*parent)
	for i, page := range children {
		if page.RenderedPath() == child.RenderedPath() {
			if i-1 >= 0 {
				return children[i-1]
			}
		}
	}

	return nil
}
