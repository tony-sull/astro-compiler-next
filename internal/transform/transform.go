package transform

import (
	"fmt"
	"strings"

	tycho "github.com/snowpackjs/astro/internal"
	"golang.org/x/net/html/atom"
	a "golang.org/x/net/html/atom"
)

type TransformOptions struct {
	As              string
	Scope           string
	Filename        string
	InternalURL     string
	SourceMap       string
	Site            string
	PreprocessStyle interface{}
}

func Transform(doc *tycho.Node, opts TransformOptions) *tycho.Node {
	shouldScope := len(doc.Styles) > 0 && ScopeStyle(doc.Styles, opts)
	walk(doc, func(n *tycho.Node) {
		ExtractScript(doc, n)
		if shouldScope {
			ScopeElement(n, opts)
		}
	})
	return doc
}

func ExtractStyles(doc *tycho.Node) {
	walk(doc, func(n *tycho.Node) {
		if n.Type == tycho.ElementNode {
			switch n.DataAtom {
			case a.Style:
				// Do not extract <style> inside of SVGs
				if n.Parent != nil && n.Parent.DataAtom == atom.Svg {
					return
				}
				doc.Styles = append(doc.Styles, n)
				// Remove local style node
				n.Parent.RemoveChild(n)
			}
		}
	})
}

func ExtractScript(doc *tycho.Node, n *tycho.Node) {
	if n.Type == tycho.ElementNode {
		switch n.DataAtom {
		case a.Script:
			// if <script hoist>, hoist to the document root
			if hasTruthyAttr(n, "hoist") {
				doc.Scripts = append(doc.Scripts, n)
				// Remove local script node
				n.Parent.RemoveChild(n)
			}
			// otherwise leave in place
		default:
			if n.Component || n.CustomElement {
				for _, attr := range n.Attr {
					id := n.Data
					if n.CustomElement {
						id = fmt.Sprintf("'%s'", id)
					}

					if strings.HasPrefix(attr.Key, "client:") && attr.Key != "client:only" {
						doc.HydratedComponents = append(doc.HydratedComponents, n)
						pathAttr := tycho.Attribute{
							Key:  "client:component-path",
							Val:  fmt.Sprintf("$$metadata.getPath(%s)", id),
							Type: tycho.ExpressionAttribute,
						}
						n.Attr = append(n.Attr, pathAttr)

						exportAttr := tycho.Attribute{
							Key:  "client:component-export",
							Val:  fmt.Sprintf("$$metadata.getExport(%s)", id),
							Type: tycho.ExpressionAttribute,
						}
						n.Attr = append(n.Attr, exportAttr)
						break
					}
				}
			}
		}
	}
}

func walk(doc *tycho.Node, cb func(*tycho.Node)) {
	var f func(*tycho.Node)
	f = func(n *tycho.Node) {
		cb(n)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}
