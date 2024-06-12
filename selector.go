package colibri

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	KeyAll = "all"

	KeyExpr = "expr"

	KeyFollow = "follow"

	KeyName = "name"

	KeyType = "type"
)

var (
	// ErrInvalidSelector is returned when the value is not a valid selector.
	ErrInvalidSelector = errors.New("invalid selector")

	// ErrInvalidSelectors is returned when the value is not a valid selector value.
	ErrInvalidSelectors = errors.New("invalid selectors")
)

var selectorPool = sync.Pool{
	New: func() any {
		return &Selector{Extra: make(map[string]any)}
	},
}

type Selector struct {
	// Name selector name.
	Name string

	// Expr stores the selector expression.
	Expr string

	// Type stores the type of the selector expression.
	Type string

	// All specifies whether all elements are to be found.
	All bool

	// Follow specifies whether the URLs found by the selector should be followed.
	Follow bool

	// Method specifies the HTTP method (GET, POST, PUT, ...).
	Method string

	// Proxy specifies the URL of the proxy.
	Proxy *url.URL

	// Header contains the HTTP header.
	Header http.Header

	// Timeout specifies the time limit for the HTTP request.
	Timeout time.Duration

	// Selectors nested selectors.
	Selectors []*Selector

	// Extra stores additional data.
	Extra map[string]any
}

func newSelector(name string, rawSelector any) (*Selector, error) {
	var (
		selector = selectorPool.Get().(*Selector)
		err      error
	)

	switch selectorValue := rawSelector.(type) {
	case string:
		selector.Expr = selectorValue

	case map[string]any:
		selector.Extra = selectorValue
		err = processRaw(selector.Extra, selector)

	default:
		return nil, ErrInvalidSelector
	}

	selector.Name = name
	return selector, err
}

func newSelectors(rawSelectors any) ([]*Selector, error) {
	if rawSelectors == nil {
		return nil, nil
	}

	selectorsMap, ok := rawSelectors.(map[string]any)
	if !ok {
		return nil, ErrInvalidSelectors
	}

	var (
		selectors []*Selector
		errs      error
	)
	for name, value := range selectorsMap {
		if (name == "") || (value == nil) {
			continue
		}

		selector, err := newSelector(name, value)
		if err != nil {
			errs = AddError(errs, name, err)
		} else if selector != nil {
			selectors = append(selectors, selector)
		}
	}
	return selectors, errs
}

// Rules returns a Rules with the Selector's data.
//
// If the selector does not have a specified value for the Proxy, User-Agent, or Timeout fields,
// the values from the source rules are used.
//
// The values for the Cookies, IgnoreRobotsTxt, Delay, Redirects, ResponseBodySize fields are obtained from the source rules.
func (sel *Selector) Rules(src *Rules) *Rules {
	newRules := rulesPool.Get().(*Rules)

	newRules.Method = sel.Method

	if sel.Proxy != nil {
		newRules.Proxy = sel.Proxy.ResolveReference(&url.URL{})
	} else if src.Proxy != nil {
		newRules.Proxy = src.Proxy.ResolveReference(&url.URL{})
	}

	if sel.Header != nil {
		newRules.Header = sel.Header.Clone()

	} else if src.Header != nil {
		newRules.Header = http.Header{}

		if ua := src.Header.Get("User-Agent"); ua != "" {
			newRules.Header.Set("User-Agent", ua)
		}
	}

	if sel.Timeout == 0 {
		newRules.Timeout = src.Timeout
	} else if sel.Timeout > 0 {
		newRules.Timeout = sel.Timeout
	}

	newRules.Cookies = src.Cookies
	newRules.IgnoreRobotsTxt = src.IgnoreRobotsTxt
	newRules.Delay = src.Delay
	newRules.Redirects = src.Redirects
	newRules.ResponseBodySize = src.ResponseBodySize

	if len(sel.Selectors) > 0 {
		newRules.Selectors = CloneSelectors(sel.Selectors)
	}

	newRules.Extra = make(map[string]any)
	for key, value := range sel.Extra {
		newRules.Extra[key] = value
	}

	return newRules
}

// Clone returns a copy of the original selector.
//
// Cloning the Extra field can cause errors, so you should avoid storing pointers.
func (sel *Selector) Clone() *Selector {
	newSelector := selectorPool.Get().(*Selector)

	newSelector.Name = sel.Name
	newSelector.Expr = sel.Expr
	newSelector.Type = sel.Type
	newSelector.All = sel.All
	newSelector.Follow = sel.Follow

	newSelector.Method = sel.Method

	if sel.Proxy != nil {
		newSelector.Proxy = sel.Proxy.ResolveReference(&url.URL{})
	}

	newSelector.Header = sel.Header.Clone()
	newSelector.Timeout = sel.Timeout

	if len(sel.Selectors) > 0 {
		newSelector.Selectors = CloneSelectors(sel.Selectors)
	}

	newSelector.Extra = make(map[string]any)
	for key, value := range sel.Extra {
		newSelector.Extra[key] = value
	}
	return newSelector
}

// Clear clears all fields in the selector.
//
// Selectors are released, see the ReleaseSelector function.
func (sel *Selector) Clear() {
	sel.Name = ""
	sel.Expr = ""
	sel.Type = ""
	sel.All = false
	sel.Follow = false

	sel.Method = ""
	sel.Proxy = nil
	sel.Header = nil
	sel.Timeout = 0

	sel.Selectors = ReleaseSelectors(sel.Selectors)
	clear(sel.Extra)
}

// ReleaseRules clears and sends the selector to the selector pool.
func ReleaseSelector(selector *Selector) {
	selector.Clear()
	selectorPool.Put(selector)
}

func ReleaseSelectors(selectors []*Selector) []*Selector {
	for _, selector := range selectors {
		ReleaseSelector(selector)
	}
	return nil
}

// CloneSelectors clones the selectors.
func CloneSelectors(selectors []*Selector) []*Selector {
	result := make([]*Selector, 0, len(selectors))
	for _, selector := range selectors {
		result = append(result, selector.Clone())
	}
	return result
}
