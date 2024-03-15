// parsers is an interface that Colibri can use to parse the content of responses.
package parsers

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sync"

	"github.com/gonzxlez/colibri"
)

const (
	XPathExpr = "xpath"

	CSSelector = "css"

	RegularExpr = "regular"
)

var (
	// ErrNotMatch is returned when Content-Tyepe does not match Paser.
	ErrNotMatch = errors.New("Content-Type does not match")

	// ErrExprType is returned when the expression type is not supported by the node.
	ErrExprType = errors.New("ExprType not compatible with node")
)

type Node interface {
	// Find finds the first child node that matches the expression.
	Find(expr, exprType string) (Node, error)

	// FindAll finds all child nodes that match the expression.
	FindAll(expr, exprType string) ([]Node, error)

	// Value returns the value of the node.
	Value() any
}

// Parsers is used to parse the content of the answers.
// When a regular expression matches the content type of the response, the content
// of the response is parsed with the parser corresponding to the regular expression.
type Parsers struct {
	rw    sync.RWMutex
	funcs map[string]*parser
}

type parser struct {
	RE   *regexp.Regexp
	Func func(colibri.Response) (Node, error)
}

// New returns a new default parser to parse HTML, XHML, JSON, and plain text.
// See the colibri.Parser interface.
func New() (*Parsers, error) {
	parsers := &Parsers{
		funcs: make(map[string]*parser),
	}

	var errs error
	errs = colibri.AddError(errs, "HTML", Set(parsers, HTMLRegexp, ParseHTML))
	errs = colibri.AddError(errs, "JSON", Set(parsers, JSONRegexp, ParseJSON))
	errs = colibri.AddError(errs, "TEXT", Set(parsers, TextRegexp, ParseText))
	errs = colibri.AddError(errs, "XML", Set(parsers, XMLRegexp, ParseXML))

	return parsers, errs
}

// Set adds a parser with its regular expression corresponding to the parsers.
func Set[T Node](parsers *Parsers, expr string, parserFunc func(colibri.Response) (T, error)) error {
	if (parsers == nil) || (expr == "") || (parserFunc == nil) {
		return nil
	}

	regular, err := regexp.Compile(expr)
	if err != nil {
		return err
	}

	parsers.rw.Lock()
	parsers.funcs[expr] = &parser{
		RE: regular,
		Func: func(resp colibri.Response) (Node, error) {
			return parserFunc(resp)
		},
	}
	parsers.rw.Unlock()
	return nil
}

// Match returns true if the content-type is supported.
func (parsers *Parsers) Match(contentType string) bool {
	parsers.rw.RLock()
	defer parsers.rw.RUnlock()

	for _, p := range parsers.funcs {
		if p.RE.MatchString(contentType) {
			return true
		}
	}
	return false
}

// Parse parses the response based on the rules.
func (parsers *Parsers) Parse(rules *colibri.Rules, resp colibri.Response) (map[string]any, error) {
	if (rules == nil) || (resp == nil) {
		return nil, nil
	}

	var (
		contentType = resp.Header().Get("Content-Type")
		parserFunc  func(colibri.Response) (Node, error)
	)

	parsers.rw.Lock()
	for _, p := range parsers.funcs {
		if p.RE.MatchString(contentType) {
			parserFunc = p.Func
			break
		}
	}
	parsers.rw.Unlock()

	if parserFunc == nil {
		return nil, ErrNotMatch
	}

	parent, err := parserFunc(resp)
	if err != nil {
		return nil, err
	}
	return parsers.findSelectors(rules, resp, parent)
}

func (parsers *Parsers) Clear() {
	parsers.rw.Lock()
	clear(parsers.funcs)
	parsers.rw.Unlock()
}

func (parsers *Parsers) findSelectors(rules *colibri.Rules, resp colibri.Response, parent Node) (map[string]any, error) {
	if (resp == nil) || (parent == nil) {
		return nil, nil
	}

	var (
		result = make(map[string]any)
		errs   error
	)
	for _, selector := range rules.Selectors {
		found, err := parsers.findSelector(rules, resp, selector, parent)
		if err != nil {
			errs = colibri.AddError(errs, selector.Name, err)
			continue
		}
		result[selector.Name] = found
	}
	return result, errs
}

func (parsers *Parsers) findSelector(src *colibri.Rules, resp colibri.Response, selector *colibri.Selector, parent Node) (any, error) {
	if selector.All {
		return parsers.findAllSelector(src, resp, selector, parent)
	}

	child, err := parent.Find(selector.Expr, selector.Type)
	if err != nil {
		return nil, err
	} else if child == nil {
		return nil, nil
	}

	if selector.Follow {
		rules := selector.Rules(src)
		defer colibri.ReleaseRules(rules)

		return parsers.followSelector(rules, resp, child.Value())
	}

	if len(selector.Selectors) > 0 {
		rules := selector.Rules(src)
		defer colibri.ReleaseRules(rules)

		return parsers.findSelectors(rules, resp, child)
	}
	return child.Value(), nil
}

func (parsers *Parsers) findAllSelector(src *colibri.Rules, resp colibri.Response, selector *colibri.Selector, parent Node) (any, error) {
	children, err := parent.FindAll(selector.Expr, selector.Type)
	if err != nil {
		return nil, err
	}

	var (
		result []any
		errs   error
	)
	if !selector.Follow && (len(selector.Selectors) > 0) {
		rules := selector.Rules(src)
		defer colibri.ReleaseRules(rules)

		for i, child := range children {
			found, err := parsers.findSelectors(rules, resp, child)
			if err != nil {
				errs = colibri.AddError(errs, fmt.Sprintf("%s+#%d", selector.Name, i), err)
				continue
			}
			result = append(result, found)
		}

		return result, errs
	}

	for _, child := range children {
		result = append(result, child.Value())
	}

	if selector.Follow {
		rules := selector.Rules(src)
		defer colibri.ReleaseRules(rules)

		return parsers.followSelector(rules, resp, result...)
	}
	return result, errs
}

func (parsers *Parsers) followSelector(rules *colibri.Rules, resp colibri.Response, rawURL ...any) ([]map[string]any, error) {
	var (
		urls = make([]*url.URL, 0, len(rawURL))
		errs error
	)

	for _, rawU := range rawURL {
		u, err := colibri.ToURL(rawU)
		if err != nil {
			errs = colibri.AddError(errs, fmt.Sprintf("%v", rawU), err)
			continue
		}

		if !u.IsAbs() {
			u = resp.URL().ResolveReference(u)
		}
		urls = append(urls, u)
	}

	if errs != nil {
		return nil, errs
	}

	var result []map[string]any
	for _, u := range urls {
		cRules := rules.Clone()
		cRules.URL = u

		out, err := resp.Extract(cRules)
		if err != nil {
			errs = colibri.AddError(errs, u.String(), err)
			continue
		}

		result = append(result, out.Serializable())
		colibri.ReleaseRules(cRules)
	}

	return result, errs
}
