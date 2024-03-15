package parsers

import (
	"strings"

	"github.com/gonzxlez/colibri"

	"github.com/andybalholm/cascadia"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// HTMLRegexp contains a regular expression that matches the HTML MIME type.
const HTMLRegexp = `^text\/html`

type HTMLNode struct {
	node *html.Node
}

func ParseHTML(resp colibri.Response) (*HTMLNode, error) {
	contentType := resp.Header().Get("Content-Type")
	r, err := charset.NewReader(resp.Body(), contentType)
	if err != nil {
		return nil, err
	}

	root, err := htmlquery.Parse(r)
	if err != nil {
		return nil, err
	}
	return &HTMLNode{root}, nil
}

func (html *HTMLNode) Find(expr, exprType string) (Node, error) {
	if exprType == "" {
		exprType = XPathExpr
	}

	switch {
	case strings.EqualFold(exprType, XPathExpr):
		return html.XPathFind(expr)
	case strings.EqualFold(exprType, CSSelector):
		return html.CSSFind(expr)
	}
	return nil, ErrExprType
}

func (html *HTMLNode) FindAll(expr, exprType string) ([]Node, error) {
	if exprType == "" {
		exprType = XPathExpr
	}

	switch {
	case strings.EqualFold(exprType, XPathExpr):
		return html.XPathFindAll(expr)
	case strings.EqualFold(exprType, CSSelector):
		return html.CSSFindAll(expr)
	}
	return nil, ErrExprType
}

func (html *HTMLNode) Value() any {
	return htmlquery.InnerText(html.node)
}

func (html *HTMLNode) XPathFind(expr string) (Node, error) {
	htmlNode, err := htmlquery.Query(html.node, expr)
	if err != nil {
		return nil, err
	} else if htmlNode == nil {
		return nil, nil
	}

	return &HTMLNode{htmlNode}, nil
}

func (html *HTMLNode) XPathFindAll(expr string) ([]Node, error) {
	htmlNodes, err := htmlquery.QueryAll(html.node, expr)
	if err != nil {
		return nil, err
	}

	var elements []Node
	for _, node := range htmlNodes {
		elements = append(elements, &HTMLNode{node})
	}
	return elements, nil
}

func (html *HTMLNode) CSSFind(expr string) (Node, error) {
	sel, err := cascadia.Compile(expr)
	if err != nil {
		return nil, err
	}

	htmlNode := cascadia.Query(html.node, sel)
	if htmlNode == nil {
		return nil, nil
	}
	return &HTMLNode{htmlNode}, nil
}

func (html *HTMLNode) CSSFindAll(expr string) ([]Node, error) {
	sel, err := cascadia.Compile(expr)
	if err != nil {
		return nil, err
	}

	var elements []Node
	for _, node := range cascadia.QueryAll(html.node, sel) {
		elements = append(elements, &HTMLNode{node})
	}
	return elements, nil
}
