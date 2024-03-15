package webextractor

import (
	"io"
	"net/http"
	"net/url"

	"github.com/gonzxlez/colibri"
)

// Response represents an HTTP response.
// See the colibri.Response interface.
type Response struct {
	HTTP      *http.Response
	redirects []*url.URL
	c         *colibri.Colibri
}

func (resp *Response) URL() *url.URL {
	return resp.HTTP.Request.URL
}

func (resp *Response) StatusCode() int {
	return resp.HTTP.StatusCode
}

func (resp *Response) Header() http.Header {
	return resp.HTTP.Header
}

func (resp *Response) Body() io.ReadCloser {
	return resp.HTTP.Body
}

func (resp *Response) Redirects() []*url.URL {
	return resp.redirects
}

func (resp *Response) Serializable() map[string]any {
	var redirects []string
	for _, u := range resp.Redirects() {
		redirects = append(redirects, u.String())
	}

	return map[string]any{
		"url":       resp.HTTP.Request.URL.String(),
		"code":      resp.HTTP.StatusCode,
		"header":    resp.HTTP.Header,
		"redirects": redirects,
	}
}

func (resp *Response) Do(rules *colibri.Rules) (colibri.Response, error) {
	return resp.c.Do(rules)
}

func (resp *Response) Extract(rules *colibri.Rules) (*colibri.Output, error) {
	return resp.c.Extract(rules)
}
