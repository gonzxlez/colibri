package colibri

import (
	"encoding/json"
	"net/http"
	"net/url"
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

	KeyResponseBodySize = "responseBodySize"

	KeySelectors = "selectors"

	KeyTimeout = "timeout"

	KeyURL = "URL"
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

	// ResponseBodySize maximum response body size.
	ResponseBodySize int

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
	newRules.ResponseBodySize = rules.ResponseBodySize

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
	rules.ResponseBodySize = 0

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
