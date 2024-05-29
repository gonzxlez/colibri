// parsers is an interface that Colibri can use to parse the content of responses.
package parsers

import (
	"errors"
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

// Parsers is used to parse the content of the answers.
// When a regular expression matches the content type of the response, the content
// of the response is parsed with the parser corresponding to the regular expression.
type Parsers struct {
	rw    sync.RWMutex
	funcs map[string]*parser
}

type parser struct {
	RE   *regexp.Regexp
	Func func(colibri.Response) (colibri.Node, error)
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
func Set[T colibri.Node](parsers *Parsers, expr string, parserFunc func(colibri.Response) (T, error)) error {
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
		Func: func(resp colibri.Response) (colibri.Node, error) {
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
func (parsers *Parsers) Parse(rules *colibri.Rules, resp colibri.Response) (colibri.Node, error) {
	if (rules == nil) || (resp == nil) {
		return nil, nil
	}

	var (
		contentType = resp.Header().Get("Content-Type")
		parserFunc  func(colibri.Response) (colibri.Node, error)
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

	return parserFunc(resp)
}

func (parsers *Parsers) Clear() {
	parsers.rw.Lock()
	clear(parsers.funcs)
	parsers.rw.Unlock()
}
