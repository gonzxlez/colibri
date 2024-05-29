package parsers

import (
	"strings"

	"github.com/gonzxlez/colibri"

	"github.com/antchfx/jsonquery"
)

// JSONRegexp contains a regular expression that matches the JSON MIME type.
const JSONRegexp = `^application\/(json|x-json|([a-z]+\+json))`

type JSONode struct {
	node *jsonquery.Node
}

func ParseJSON(resp colibri.Response) (*JSONode, error) {
	root, err := jsonquery.Parse(resp.Body())
	if err != nil {
		return nil, err
	}
	return &JSONode{root}, nil
}

func (json *JSONode) Find(selector *colibri.Selector) (colibri.Node, error) {
	if (selector.Type != "") && !strings.EqualFold(selector.Type, XPathExpr) {
		return nil, ErrExprType
	}

	jsonNode, err := jsonquery.Query(json.node, selector.Expr)
	if err != nil {
		return nil, err
	} else if jsonNode == nil {
		return nil, nil
	}

	return &JSONode{jsonNode}, nil
}

func (json *JSONode) FindAll(selector *colibri.Selector) ([]colibri.Node, error) {
	if (selector.Type != "") && !strings.EqualFold(selector.Type, XPathExpr) {
		return nil, ErrExprType
	}

	jsonNodes, err := jsonquery.QueryAll(json.node, selector.Expr)
	if err != nil {
		return nil, err
	}

	var nodes []colibri.Node
	for _, node := range jsonNodes {
		nodes = append(nodes, &JSONode{node})
	}
	return nodes, nil
}

func (json *JSONode) Value() any {
	return json.node.Value()
}
