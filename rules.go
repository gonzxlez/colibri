package colibri

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	KeyCookies = "cookies"

	KeyDelay = "delay"

	KeyHeader = "header"

	KeyIgnoreRobotsTxt = "ignoreRobotsTxt"

	KeyMethod = "method"

	KeyProxy = "proxy"

	KeyRedirects = "redirects"

	KeySelectors = "selectors"

	KeyTimeout = "timeout"

	KeyURL = "URL"
)

var (
	// ErrInvalidHeader is returned when the value is an invalid header.
	ErrInvalidHeader = errors.New("invalid header")

	// ErrMustBeString is returned when the value is not a string.
	ErrMustBeString = errors.New("must be a string")

	// ErrMustBeNumber is returned when the value is not a number.
	ErrMustBeNumber = errors.New("must be a number")

	// ErrNotAssignable is returned when the value is not assignable to the field.
	ErrNotAssignable = errors.New("value is not assignable to field")
)

var (
	intType = reflect.TypeOf(int(0))

	urlType = reflect.TypeOf((*url.URL)(nil))

	headerType = reflect.TypeOf(http.Header{})

	durationType = reflect.TypeOf(time.Duration(0))

	selectorsType = reflect.TypeOf([]*Selector{})
)

var rulesPool = sync.Pool{
	New: func() any {
		return &Rules{Extra: make(map[string]any)}
	},
}

type Rules struct {
	//  Method specifies the HTTP method (GET, POST, PUT, ...).
	Method string

	// URL specifies the URL of the request.
	URL *url.URL

	// Proxy specifies the URL of the proxy.
	Proxy *url.URL

	// Header contains the HTTP header.
	Header http.Header

	// Timeout specifies the time limit for the HTTP request.
	Timeout time.Duration

	// Cookies specifies whether the client should send and store Cookies.
	Cookies bool

	// IgnoreRobotsTxt specifies whether robots.txt should be ignored.
	IgnoreRobotsTxt bool

	// Delay specifies the delay time between requests.
	Delay time.Duration

	// Redirects specifies the maximum number of redirects.
	Redirects int

	// Selectors
	Selectors []*Selector

	// Extra stores additional data.
	Extra map[string]any
}

// Clone returns a copy of the original rules.
//
// Cloning the Extra field can cause errors, so you should avoid storing pointers.
func (rules *Rules) Clone() *Rules {
	newRules := rulesPool.Get().(*Rules)

	if rules.URL != nil {
		newRules.URL = rules.URL.ResolveReference(&url.URL{})
	}

	if rules.Proxy != nil {
		newRules.Proxy = rules.Proxy.ResolveReference(&url.URL{})
	}

	newRules.Method = rules.Method
	newRules.Header = rules.Header.Clone()
	newRules.Timeout = rules.Timeout
	newRules.Cookies = rules.Cookies
	newRules.IgnoreRobotsTxt = rules.IgnoreRobotsTxt
	newRules.Delay = rules.Delay
	newRules.Redirects = rules.Redirects

	if len(rules.Selectors) > 0 {
		newRules.Selectors = CloneSelectors(rules.Selectors)
	}

	newRules.Extra = make(map[string]any)
	for key, value := range rules.Extra {
		newRules.Extra[key] = value
	}
	return newRules
}

// Clear clears all fields from the rules.
//
// Selectors are released, see the ReleaseSelector function.
func (rules *Rules) Clear() {
	rules.Method = ""
	rules.URL = nil
	rules.Proxy = nil
	rules.Header = nil
	rules.Timeout = 0
	rules.Cookies = false
	rules.IgnoreRobotsTxt = false
	rules.Delay = 0
	rules.Redirects = 0

	rules.Selectors = ReleaseSelectors(rules.Selectors)
	clear(rules.Extra)
}

func (rules *Rules) UnmarshalJSON(b []byte) (err error) {
	newRules := rulesPool.Get().(*Rules)

	if err := json.Unmarshal(b, &newRules.Extra); err != nil {
		return err
	}

	if err := processRaw(newRules.Extra, newRules); err != nil {
		return err
	}

	*rules = *newRules
	return nil
}

// ReleaseRules clears and sends the rules to the rules pool.
func ReleaseRules(rules *Rules) {
	rules.Clear()
	rulesPool.Put(rules)
}

func processRaw[T Rules | Selector](raw map[string]any, output *T) error {
	if raw == nil {
		return nil
	}

	var (
		rOutput = reflect.ValueOf(output).Elem()
		err     error
		errs    error
	)

	for key, value := range raw {
		field := rOutput.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(key, s)
		})

		if field.IsValid() && field.CanSet() {
			fieldType := field.Type()

			switch fieldType {
			case intType:
				value, err = toInt(value)
			case urlType:
				value, err = ToURL(value)
			case headerType:
				value, err = toHeader(value)
			case durationType:
				value, err = toDuration(value)
			case selectorsType:
				value, err = newSelectors(value)
			}

			if err != nil {
				errs = AddError(errs, key, err)
				continue
			}

			rValue := reflect.ValueOf(value)
			if !rValue.Type().AssignableTo(fieldType) {
				errs = AddError(errs, key, ErrNotAssignable)
				continue
			}

			field.Set(rValue)
			delete(raw, key)
		}
	}

	return errs
}

// ToURL converts a value to a *url.URL.
func ToURL(value any) (*url.URL, error) {
	rawURL, ok := value.(string)
	if ok {
		return url.Parse(rawURL)
	}
	return nil, ErrMustBeString
}

func toInt(value any) (int, error) {
	switch n := value.(type) {
	case int:
		return n, nil
	case float64:
		return int(n), nil
	}
	return 0, ErrMustBeNumber
}

func toHeader(value any) (http.Header, error) {
	header := http.Header{}

	if value == nil {
		return header, nil
	}

	headerMap, ok := value.(map[string]any)
	if !ok {
		return header, ErrInvalidHeader
	}

	for k, v := range headerMap {
		switch val := v.(type) {
		case string:
			header.Add(k, val)
		case []any:
			for _, e := range val {
				s, ok := e.(string)
				if !ok {
					return header, ErrInvalidHeader
				}
				header.Add(k, s)
			}

		default:
			return header, ErrInvalidHeader
		}
	}
	return header, nil
}

func toDuration(value any) (time.Duration, error) {
	switch d := value.(type) {
	case int:
		return time.Duration(d) * time.Millisecond, nil
	case float64:
		return time.Duration(d*1000000) * time.Nanosecond, nil
	}
	return 0, ErrMustBeNumber
}
