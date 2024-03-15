package webextractor

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gonzxlez/colibri"
)

const (
	gotWantFormat       = "got %v, want %v"
	prefixGotWantFormat = "%v: got %v, want %v"
)

func mustNewURL(rawURL string) *url.URL {
	u, _ := url.Parse(rawURL)
	return u
}

func TestExtract(t *testing.T) {
	ts := testServer()
	defer ts.Close()

	we, err := New()
	if err != nil {
		t.Fatal(err)
	}

	var (
		rawRules = []byte(`{
			"Delay": 5,
			"Selectors": {
				"html": {
					"Expr":   "\/html",
					"Type":   "regular",
					"Follow": true,
					"Selectors": {
						"title": "//title",
						"links": {
							"Expr": "//a/@href",
							"Type": "xpath",
							"All":  true
						}
					}
				},

				"xml": {
					"Expr":   "\/xml",
					"Type":   "regular",
					"Follow": true,
					"Selectors": {
						"title": "//title",

						"json": {
							"Expr":   "//json",
							"Type":   "xpath",
							"Follow": true,
							"Selectors": {
								"text": "//text"
							}
						}
					}
				}
			}
		}`)

		emptySlice []string

		wantOutput = map[string]any{
			"html": []map[string]any{
				{
					"response": map[string]any{
						"url":       ts.URL + "/html",
						"code":      200,
						"redirects": emptySlice,
						"header": http.Header{
							"Content-Type":   []string{"text/html"},
							"Date":           []string{""},
							"Content-Length": []string{"193"},
						},
					},
					"data": map[string]any{
						"title": "My test page",
						"links": []any{"/json", "/text", "/xml"},
					},
				},
			},
			"xml": []map[string]any{
				{
					"response": map[string]any{
						"url":       ts.URL + "/xml",
						"code":      200,
						"redirects": emptySlice,
						"header": http.Header{
							"Content-Type":   []string{"application/xml"},
							"Date":           []string{""},
							"Content-Length": []string{"126"},
						},
					},
					"data": map[string]any{
						"title": "XML Doc",

						"json": []map[string]any{
							{
								"response": map[string]any{
									"url":       ts.URL + "/json",
									"code":      200,
									"redirects": emptySlice,
									"header": http.Header{
										"Content-Type":   []string{"application/json"},
										"Date":           []string{""},
										"Content-Length": []string{"59"},
									},
								},
								"data": map[string]any{
									"text": "/text",
								},
							},
						},
					},
				},
			},
		}
	)

	rules := &colibri.Rules{}
	defer colibri.ReleaseRules(rules)

	if err := json.Unmarshal(rawRules, rules); err != nil {
		t.Fatal(err)
	}

	rules.URL = mustNewURL(ts.URL + "/text")
	output, err := we.Extract(rules)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ResponseExtract", func(t *testing.T) {
		output2, err := output.Response.Extract(rules)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(output2.Data, wantOutput) {
			t.Fatal("not equal")
		}
	})

	if !reflect.DeepEqual(output.Data, wantOutput) {
		t.Fatal("not equal")
	}
}

func TestCookies(t *testing.T) {
	ts := testServerCookies()
	defer ts.Close()

	we, err := New()
	if err != nil {
		t.Fatal(err)
	}
	we.Delay = nil     // Deactivate Delay
	we.RobotsTxt = nil // Deactivate RobotsTxt

	jar := we.Client.(*Client).Jar

	tests := []struct {
		Path           string
		Cookies        bool
		WantLenCookies int
		WantStatusCode int
	}{
		{"/set", false, 0, http.StatusOK},
		{"/set", true, 1, http.StatusOK},
		{"/check", false, 1, http.StatusInternalServerError},
		{"/check", true, 1, http.StatusOK},
	}

	for _, tt := range tests {
		var (
			name = strconv.FormatBool(tt.Cookies) + tt.Path

			rules = &colibri.Rules{
				Method:  "GET",
				URL:     mustNewURL(ts.URL + tt.Path),
				Cookies: tt.Cookies,
			}
		)

		t.Run(name, func(t *testing.T) {
			if resp, err := we.Do(rules); err != nil {
				t.Fatal(err)
			} else if resp.StatusCode() != tt.WantStatusCode {
				t.Fatalf(prefixGotWantFormat, "Status Code", resp.StatusCode(), tt.WantStatusCode)
			}

			cookies := jar.Cookies(rules.URL)
			if lenCookies := len(cookies); lenCookies != tt.WantLenCookies {
				t.Fatalf(prefixGotWantFormat, "LenCookies", lenCookies, tt.WantLenCookies)
			}
		})
	}

	t.Run("NewClientWithJar", func(t *testing.T) {
		u := mustNewURL(ts.URL)
		wantLenCookies := len(jar.Cookies(u))

		we2, err := New(jar)
		if err != nil {
			t.Fatal(err)
		}

		jar2 := we2.Client.(*Client).Jar

		cookies := jar2.Cookies(u)
		if len(cookies) != wantLenCookies {
			t.Fatal("Number of unexpected cookies")
		}
	})

	t.Run("ClientClear", func(t *testing.T) {
		client := we.Client.(*Client)
		if client.Jar == nil {
			t.Fatal("Nil Jar")
		}

		we.Clear()

		if client.Jar != nil {
			t.Fatal("Jar must be nil")
		}
	})
}

func TestUserAgent(t *testing.T) {
	ts := testServer()
	defer ts.Close()

	we, err := New()
	if err != nil {
		t.Fatal(err)
	}

	we.Delay = nil     // Deactivate Delay
	we.RobotsTxt = nil // Deactivate RobotsTxt

	var doer interface {
		Do(rules *colibri.Rules) (colibri.Response, error)
	} = we

	tests := []struct {
		UserAgent      string
		WantStatusCode int
		WantUserAgent  string
	}{
		{"", http.StatusOK, colibri.DefaultUserAgent},
		{"test/req2", http.StatusOK, "test/req2"},
		{"test/req3", http.StatusOK, "test/req3"},
	}

	for _, tt := range tests {
		var (
			name = "User-Agent(" + tt.WantUserAgent + ")"

			rules = &colibri.Rules{
				Method: "GET",
				URL:    mustNewURL(ts.URL),
				Header: http.Header{"User-Agent": []string{tt.UserAgent}},
			}
		)

		t.Run(name, func(t *testing.T) {
			resp, err := doer.Do(rules)
			if err != nil {
				t.Fatal(err)
			} else if resp.StatusCode() != tt.WantStatusCode {
				t.Fatalf(prefixGotWantFormat, "Status Code", resp.StatusCode(), http.StatusOK)
			}

			reqDump, err := http.ReadRequest(bufio.NewReader(resp.Body()))
			if err != nil {
				t.Fatal(err)
			}

			if reqDump.UserAgent() != tt.WantUserAgent {
				t.Fatalf(prefixGotWantFormat, "User-Agent", reqDump.UserAgent(), tt.WantUserAgent)
			}

			doer = resp
		})
	}
}

func TestWithRobotsTxt(t *testing.T) {
	ts := testServer()
	defer ts.Close()

	we, err := New()
	if err != nil {
		t.Fatal(err)
	}

	we.Delay = nil // Deactivate Delay

	header := http.Header{"User-Agent": []string{"test/0.1"}}

	tests := []struct {
		Method          string
		Path            string
		Header          http.Header
		IgnoreRobotsTxt bool

		WantErr error
	}{
		{"GET", "", header, false, nil /*WantErr*/},
		{"POST", "/disallow", header, false, colibri.ErrorRobotstxtRestriction},
		{"PUT", "/disallow", nil, false, colibri.ErrorRobotstxtRestriction},
		{"GET", "/robots.txt", header, false, nil /*WantErr*/}, // ignore

		{"POST", "/disallow", header, true, nil /*WantErr*/},
	}

	for _, tt := range tests {
		var (
			name = strconv.FormatBool(tt.IgnoreRobotsTxt) + tt.Path

			rules = &colibri.Rules{
				Method:          tt.Method,
				URL:             mustNewURL(ts.URL + tt.Path),
				Header:          tt.Header,
				IgnoreRobotsTxt: tt.IgnoreRobotsTxt,
			}
		)

		t.Run(name, func(t *testing.T) {
			resp, err := we.Do(rules)
			if (err != nil) || (tt.WantErr != nil) {
				if !errors.Is(err, tt.WantErr) {
					t.Fatalf(gotWantFormat, err, tt.WantErr)
				}

			} else if resp.StatusCode() != http.StatusOK {
				t.Fatalf(prefixGotWantFormat, "Status Code", resp.StatusCode(), http.StatusOK)
			}
		})
	}

	t.Run("RobotsDataClear", func(t *testing.T) {
		var (
			robots = we.RobotsTxt.(*RobotsData)
			u      = mustNewURL(ts.URL)
		)

		if _, ok := robots.data[u.Host]; !ok {
			t.Fatal("")
		}

		robots.Clear()

		if len(robots.data) > 0 {
			t.Fatal("")
		}
	})
}

func TestWithRedirects(t *testing.T) {
	ts := testServer()
	defer ts.Close()

	we, err := New()
	if err != nil {
		t.Fatal(err)
	}

	we.Delay = nil // Deactivate Delay

	tests := []struct {
		N         int
		Max       int
		Redirects []string
	}{
		{0, 0, nil},
		{0, 2, nil},
		{1, 1, []string{ts.URL + "/redirect?n=1"}},
		{3, 5, []string{ts.URL + "/redirect?n=3", ts.URL + "/redirect?n=2", ts.URL + "/redirect?n=1"}},

		{3, 2, nil},
		{1, 0, nil},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("(N%d_MAX%d)", tt.N, tt.Max), func(t *testing.T) {
			rules := &colibri.Rules{
				URL:       mustNewURL(fmt.Sprintf("%s/redirect?n=%d", ts.URL, tt.N)),
				Redirects: tt.Max,
			}

			var wantErr error
			if tt.N > tt.Max {
				wantErr = colibri.ErrMaxRedirects
			}

			resp, err := we.Do(rules)
			if !errors.Is(err, wantErr) {
				t.Fatal(err)
			}

			if wantErr != nil {
				return
			}

			respMap := resp.Serializable()
			if respMap["url"] != ts.URL+"/redirect?n=0" {
				t.Fatal("")
			} else if !reflect.DeepEqual(respMap["redirects"], tt.Redirects) {
				t.Fatal(respMap["redirects"])
			}
		})
	}
}

/* Benchmark */
func BenchmarkHTTPClient(b *testing.B) {
	ts := testServer()
	defer ts.Close()

	c := &http.Client{}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		req, err := http.NewRequest("GET", ts.URL+"/?n="+strconv.Itoa(n), nil)
		if err != nil {
			b.Fatal(err)
		}

		if _, err = c.Do(req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkColibri(b *testing.B) {
	ts := testServer()
	defer ts.Close()

	we, err := New()
	if err != nil {
		b.Fatal(err)
	}

	we.Delay = nil     // Deactivate Delay
	we.RobotsTxt = nil // Deactivate RobotsTxt

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		rules := &colibri.Rules{
			Method: "GET",
			URL:    mustNewURL(ts.URL + "/?n=" + strconv.Itoa(n)),
		}

		if _, err := we.Do(rules); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWithRandomCookies(b *testing.B) {
	ts := testServerCookies()
	defer ts.Close()

	we, err := New()
	if err != nil {
		b.Fatal(err)
	}

	we.Delay = nil     // Deactivate Delay
	we.RobotsTxt = nil // Deactivate RobotsTxt

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		rules := &colibri.Rules{
			Method:  "GET",
			URL:     mustNewURL(ts.URL + "/random"),
			Cookies: true,
		}

		if _, err := we.Do(rules); err != nil {
			b.Fatal(err)
		}
	}
}

/* httptest*/
const (
	characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	robotsTXT = `User-agent: *
	Disallow: /disallow`

	htmlBody = `<!doctype html>
	<html>
		<head>
			<title>My test page</title>
		</head>
		<body>
			<a href="/json">json</a>
  			<a href="/text">text</a>
  			<a href="/xml">xml</a>
  		</body>
	</html>	
	`

	jsonBody = `{
		"html": "/html",
		"text": "/text",
		"xml": "/xml"
	}`

	textBody = `HTML: /html
	JSON: /json
	XML: /xml`

	xmlBody = `<?xml version="1.0" encoding="UTF-8" ?>
	<title>XML Doc</title>
	<html>/html</html>
	<json>/json</json>
	<text>/text</text>
	`
)

func testServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Date", "")

		switch r.URL.Path {
		case "/":
			dump, err := httputil.DumpRequest(r, true)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(dump)
			return

		case "/disallow":
			fmt.Fprintln(w, "disallow")
			return

		case "/robots.txt":
			fmt.Fprintln(w, robotsTXT)
			return

		case "/redirect":
			var n int
			if v := r.URL.Query().Get("n"); v != "" {
				n, _ = strconv.Atoi(v)
			}

			if n > 0 {
				http.Redirect(w, r, "/redirect?n="+strconv.Itoa(n-1), http.StatusSeeOther)
			}
			return

		case "/html":
			w.Header().Add("Content-Type", "text/html")
			fmt.Fprintln(w, htmlBody)
			return

		case "/json":
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, jsonBody)
			return

		case "/text":
			w.Header().Add("Content-Type", "text/plain")
			fmt.Fprintln(w, textBody)
			return

		case "/xml":
			w.Header().Add("Content-Type", "application/xml")
			fmt.Fprintln(w, xmlBody)
			return

		default:
			http.NotFound(w, r)
		}
	}))
}

func testServerCookies() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set":
			http.SetCookie(w, &http.Cookie{Name: "Flavor", Value: "Chocolate Chip"})

		case "/check":
			if _, err := r.Cookie("Flavor"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

		case "/random":
			var b strings.Builder
			b.WriteString("cookie")

			lenCharset := len(characters)
			for i := 0; i < 4; i++ {
				b.WriteByte(characters[rand.Intn(lenCharset)])
			}
			str := b.String()

			http.SetCookie(w, &http.Cookie{Name: str, Value: str})

		default:
			http.NotFound(w, r)
		}
	}))
}
