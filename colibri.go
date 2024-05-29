// Colibri is an extensible web crawling and scraping framework for Go,
// used to crawl and extract structured data on the web.
package colibri

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultUserAgent is the default User-Agent used for requests.
const DefaultUserAgent = "colibri/0.2"

var (
	// ErrClientIsNil returned when Client is nil.
	ErrClientIsNil = errors.New("client is nil")

	// ErrParserIsNil returned when Parser is nil.
	ErrParserIsNil = errors.New("parser is nil")

	// ErrRulesIsNil returned when rules are nil.
	ErrRulesIsNil = errors.New("rules is nil")

	// ErrMaxRedirects are returned when the redirect limit is reached.
	ErrMaxRedirects = errors.New("max redirects limit reached")

	// ErrorRobotstxtRestriction is returned when the page cannot be accessed due to robots.txt restrictions.
	ErrorRobotstxtRestriction = errors.New("page not accessible due to robots.txt restriction")
)

type (
	// Response represents an HTTP response.
	Response interface {
		// URL returns the URL of the request.
		URL() *url.URL

		// StatusCode returns the status code.
		StatusCode() int

		// Header returns the HTTP header of the response.
		Header() http.Header

		// Body returns the response body.
		Body() io.ReadCloser

		// Redirects returns the redirected URLs.
		Redirects() []*url.URL

		// Serializable returns the response value as a map for easy storage or transmission.
		Serializable() map[string]any

		// Do Colibri Do method wrapper.
		// Wraps the Colibri with which the HTTP response was obtained.
		Do(rules *Rules) (Response, error)

		// Extract Colibri Extract method wrapper.
		// Wraps the Colibri with which the HTTP response was obtained.
		Extract(rules *Rules) (*Output, error)
	}

	// Client represents an HTTP client.
	Client interface {
		// Do makes HTTP requests.
		Do(c *Colibri, rules *Rules) (Response, error)

		// Clear cleans the fields of the structure.
		Clear()
	}

	// Delay manages the delay between each HTTP request.
	Delay interface {
		// Wait waits for the previous HTTP request to the same URL and stores
		// the timestamp, then starts the calculated delay with the timestamp
		// and the specified duration of the delay.
		Wait(u *url.URL, duration time.Duration)

		// Done warns that an HTTP request has been made to the URL.
		Done(u *url.URL)

		// Stamp records the time at which the HTTP request to the URL was made.
		Stamp(u *url.URL)

		// Clear cleans the fields of the structure.
		Clear()
	}

	// RobotsTxt represents a robots.txt parser.
	RobotsTxt interface {
		// IsAllowed verifies that the User-Agent can access the URL.
		IsAllowed(c *Colibri, rules *Rules) error

		// Clear cleans the fields of the structure.
		Clear()
	}

	// Parser represents a parser of the response content.
	Parser interface {
		// Match returns true if the Content-Type is supported by the parser.
		Match(contentType string) bool

		// Parse parses the response based on the rules.
		Parse(rules *Rules, resp Response) (Node, error)

		// Clear cleans the fields of the structure.
		Clear()
	}
)

type Output struct {
	// Response to Request.
	Response Response

	// Data contains the data extracted by the selectors.
	Data map[string]any
}

// Serializable returns the value of the output as a map for easy storage or transmission.
func (out *Output) Serializable() map[string]any {
	return map[string]any{
		"response": out.Response.Serializable(),
		"data":     out.Data,
	}
}

func (out *Output) MarshalJSON() ([]byte, error) {
	return json.Marshal(out.Serializable())
}

// Colibri makes HTTP requests and parses the content of the response based on rules.
type Colibri struct {
	Client    Client
	Delay     Delay
	RobotsTxt RobotsTxt
	Parser    Parser
}

// New returns a new empty Colibri structure.
func New() *Colibri {
	return &Colibri{}
}

// Do makes an HTTP request based on the rules.
func (c *Colibri) Do(rules *Rules) (resp Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	if c.Client == nil {
		return nil, ErrClientIsNil
	}

	if rules == nil {
		return nil, ErrRulesIsNil
	}

	if rules.Header == nil {
		rules.Header = http.Header{}
	}

	if rules.Header.Get("User-Agent") == "" {
		rules.Header.Set("User-Agent", DefaultUserAgent)
	}

	if (c.RobotsTxt != nil) && !rules.IgnoreRobotsTxt {
		err := c.RobotsTxt.IsAllowed(c, rules)
		if err != nil {
			return nil, err
		}
	}

	if (c.Delay != nil) && (rules.Delay > 0) {
		c.Delay.Wait(rules.URL, rules.Delay)
		defer c.Delay.Done(rules.URL)
	}

	resp, err = c.Client.Do(c, rules)

	if (c.Delay != nil) && (resp != nil) {
		c.Delay.Stamp(resp.URL())
	}
	return resp, err
}

// Extract makes the HTTP request and parses the content of the response based on the rules.
func (c *Colibri) Extract(rules *Rules) (output *Output, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	if c.Parser == nil {
		return nil, ErrParserIsNil
	}

	output = &Output{}

	output.Response, err = c.Do(rules)
	if err != nil {
		return nil, err
	}

	if len(rules.Selectors) > 0 {
		var parent Node
		parent, err = c.Parser.Parse(rules, output.Response)

		if err == nil {
			output.Data, err = FindSelectors(rules, output.Response, parent)
		}
	}
	return output, err
}

// Clear cleans the fields of the structure.
func (c *Colibri) Clear() {
	if c.Client != nil {
		c.Client.Clear()
	}

	if c.Delay != nil {
		c.Delay.Clear()
	}

	if c.RobotsTxt != nil {
		c.RobotsTxt.Clear()
	}

	if c.Parser != nil {
		c.Parser.Clear()
	}
}
