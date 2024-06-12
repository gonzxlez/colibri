// webextractor are default interfaces for Colibri ready to start crawling or extracting data on the web.
package webextractor

import (
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"

	"github.com/gonzxlez/colibri"
	"github.com/gonzxlez/colibri/webextractor/parsers"

	"golang.org/x/net/publicsuffix"
)

// New returns a new Colibri structure with default values.
// Returns an error if an error occurs when initializing the values.
func New(cookieJar ...http.CookieJar) (*colibri.Colibri, error) {
	client, err := NewClient(cookieJar...)
	if err != nil {
		return nil, err
	}

	parser, err := parsers.New()
	if err != nil {
		return nil, err
	}

	c := colibri.New()
	c.Client = client
	c.Delay = NewReqDelay()
	c.RobotsTxt = NewRobotsData()
	c.Parser = parser
	return c, nil
}

// Client represents an HTTP client.
// See the colibri.HTTPClient interface.
type Client struct {
	// Jar specifies the cookie jar.
	Jar http.CookieJar

	pool sync.Pool
}

// NewClient returns a new Client structure.
// The first cookieJar sent is taken, if no value is sent,
// a new cookiejar.Jar is initialized.
func NewClient(cookieJar ...http.CookieJar) (*Client, error) {
	client := Client{}
	if len(cookieJar) > 0 {
		client.Jar = cookieJar[0]

	} else {
		var err error
		client.Jar, err = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, err
		}
	}
	return &client, nil
}

// Do makes an HTTP request based on the rules.
func (client *Client) Do(c *colibri.Colibri, rules *colibri.Rules) (colibri.Response, error) {
	httpClient := client.getClient(rules.Proxy)
	defer client.pool.Put(httpClient)

	// CookieJar
	if rules.Cookies {
		httpClient.Jar = client.Jar
	} else {
		httpClient.Jar = nil
	}

	// Request
	req, err := httpRequest(rules)
	if err != nil {
		return nil, err
	}

	// Redirects
	var redirects []*url.URL
	httpClient.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) > rules.Redirects {
			return colibri.ErrMaxRedirects
		}

		redirects = append(redirects, via[len(via)-1].URL)
		return nil
	}

	// Response
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	r := &Response{
		HTTP:      resp,
		redirects: redirects,
		c:         c,
	}

	// ResponseBodySize
	if rules.ResponseBodySize != 0 {
		n := int64(rules.ResponseBodySize)
		r.HTTP.Body = io.NopCloser(io.LimitReader(resp.Body, n))

		if resp.ContentLength > n {
			return r, colibri.ErrResponseBodySize
		}
	}
	return r, nil
}

// Clear assigns nil to Jar.
func (client *Client) Clear() { client.Jar = nil }

func (client *Client) getClient(proxyURL *url.URL) *http.Client {
	var httpClient *http.Client
	if v := client.pool.Get(); v != nil {
		httpClient = v.(*http.Client)
	} else {
		httpClient = &http.Client{}
	}

	t, ok := httpClient.Transport.(*http.Transport)
	if (httpClient.Transport == nil) || !ok {
		t = defaultTransport()
	}

	if proxyURL != nil {
		t.Proxy = http.ProxyURL(proxyURL)
	}

	httpClient.Transport = t
	return httpClient
}

func httpRequest(rules *colibri.Rules) (*http.Request, error) {
	req, err := http.NewRequest(rules.Method, rules.URL.String(), nil /* Body */)
	if err != nil {
		return nil, err
	}
	req.Header = rules.Header
	return req, nil
}

func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		DisableKeepAlives:     true,
		MaxIdleConns:          1,
		MaxIdleConnsPerHost:   -1,
		IdleConnTimeout:       30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}
