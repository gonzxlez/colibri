package parsers

import (
	"strings"

	"github.com/gonzxlez/colibri"

	"github.com/antchfx/xmlquery"
)

// XMLRegexp contains a regular expression that matches the XML MIME type.
const XMLRegexp = `(?i)((application|image|message|model)/((\w|\.|-)+\+?)?|text/)(wb)?xml`

type XMLNode struct {
	node *xmlquery.Node
}

func ParseXML(resp colibri.Response) (*XMLNode, error) {
	root, err := xmlquery.Parse(resp.Body())
	if err != nil {
		return nil, err
	}
	return &XMLNode{root}, nil
}

func (xml *XMLNode) Find(expr, exprType string) (Node, error) {
	if (exprType != "") && !strings.EqualFold(exprType, XPathExpr) {
		return nil, ErrExprType
	}

	xmlNode, err := xmlquery.Query(xml.node, expr)
	if err != nil {
		return nil, err
	} else if xmlNode == nil {
		return nil, nil
	}

	return &XMLNode{xmlNode}, nil
}

func (xml *XMLNode) FindAll(expr, exprType string) ([]Node, error) {
	if (exprType != "") && !strings.EqualFold(exprType, XPathExpr) {
		return nil, ErrExprType
	}

	xmlNodes, err := xmlquery.QueryAll(xml.node, expr)
	if err != nil {
		return nil, err
	}

	var nodes []Node
	for _, node := range xmlNodes {
		nodes = append(nodes, &XMLNode{node})
	}
	return nodes, nil
}

func (xml *XMLNode) Value() any {
	return xml.node.InnerText()
}
