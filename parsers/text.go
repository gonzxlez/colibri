package parsers

import (
	"io"
	"regexp"
	"strings"

	"github.com/gonzxlez/colibri"
)

// TextRegexp contains a regular expression that matches the MIME type plain text.
const TextRegexp = `^text\/plain`

type TextNode struct {
	data []byte
}

func ParseText(resp colibri.Response) (*TextNode, error) {
	b, err := io.ReadAll(resp.Body())
	if err != nil {
		return nil, err
	}
	return &TextNode{b}, nil
}

func (text *TextNode) Find(expr, exprType string) (Node, error) {
	if (exprType != "") && !strings.EqualFold(exprType, RegularExpr) {
		return nil, ErrExprType
	}

	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	data := re.Find(text.data)
	return &TextNode{data}, nil
}

func (text *TextNode) FindAll(expr, exprType string) ([]Node, error) {
	if (exprType != "") && !strings.EqualFold(exprType, RegularExpr) {
		return nil, ErrExprType
	}

	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	var nodes []Node
	for _, data := range re.FindAll(text.data, -1) {
		nodes = append(nodes, &TextNode{data})
	}
	return nodes, nil
}

func (text *TextNode) Value() any {
	return string(text.data)
}
