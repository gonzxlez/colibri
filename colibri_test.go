package colibri

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"
)

var (
	testRawRulesJSON = []byte(`{
	"Method":          "GET",
	"url":             "http://example.com",
	"proxy":           "http://proxy.example.com:8080",
	"header":          {"User-Agent": "test/0.2.0"},
	"timeout":         2.5,
	"cookies":         true,
	"ignoreRobotsTXT": true,
	"delay":           1.5,
	"redirects": 3,
	"Selectors": {
		"body": {
			"name": "body",
			"expr": "//body",
			"type": "xpath",
			"all":  false,
			"selectors": {
				"urls": {
					"expr":   "//a/@href",
					"all":    true,
					"follow": true,
					"method": "get",
					"proxy":  "http://proxy.example.com:8080",
					"header": {
						"User-Agent": ["test/0.2.0"]
					},
					"timeout": 5000,
					"selectors": {
						"title": "//title"
					},
					"required": true
				}
			}
		}
	},
	"token": 505
}`)

	testBadRawRulesJSON = []byte(`{
	"Method":          0.1,
	"url":             "http://example.com",
	"proxy":           "$$$$<:>8080",
	"header":          {"User-Agent": 0.2},
	"timeout":         "2.5",
	"cookies":         "true",
	"ignoreRobotsTXT": 1,
	"delay":           {},
	"bodysize":        "5mb",
	"redirects":       true,
	"Selectors": {
		"body": {
			"name": "body",
			"expr": "//body",
			"type": "xpath",
			"all":  false,
			"selectors": {
				"urls": {
					"expr":   505,
					"all":    true,
					"follow": true,
					"method": "get",
					"proxy":  false,
					"header": {
						"User-Agent": ["test/0.2.0"]
					},
					"timeout": true,
					"selectors": "title: //title",
					"required": true
				}
			}
		}
	},
	"token": 101
}`)

	testBadRawRulesJSON_ErrInvalidSelectors = []byte(`{
	"Method":    "GET",
	"url":       "http://example.com",
	"Selectors": "err"
}`)

	testBadRawRulesJSON_ErrInvalidSelector = []byte(`{
	"Method":    "POST",
	"url":       "http://example.com",
	"Selectors": {
		"body": 101
	}
}`)
)

var (
	testSelector = &Selector{
		Name: "body",
		Expr: "//body",
		Type: "xpath",
		Selectors: []*Selector{
			{
				Name:    "urls",
				Expr:    "//a/@href",
				All:     true,
				Follow:  true,
				Method:  "get",
				Proxy:   mustNewURL("http://proxy.example.com:8080"),
				Header:  http.Header{"User-Agent": {"test/0.2.0"}},
				Timeout: 5 * time.Second,
				Selectors: []*Selector{
					{
						Name:  "title",
						Expr:  "//title",
						Extra: map[string]any{},
					},
				},
				Extra: map[string]any{
					"required": true,
				},
			},
		},
		Extra: map[string]any{},
	}

	testRules = &Rules{
		Method:          "GET",
		URL:             mustNewURL("http://example.com"),
		Proxy:           mustNewURL("http://proxy.example.com:8080"),
		Header:          http.Header{"User-Agent": {"test/0.2.0"}},
		Timeout:         2500000 * time.Nanosecond,
		Cookies:         true,
		IgnoreRobotsTxt: true,
		Delay:           1500000 * time.Nanosecond,
		Redirects:       3,
		Selectors:       []*Selector{testSelector},
		Extra: map[string]any{
			"token": float64(505),
		},
	}
)

func mustNewURL(rawURL string) *url.URL {
	u, _ := url.Parse(rawURL)
	return u
}

func TestDo(t *testing.T) {
	var (
		c      = New()
		client = &testClient{}
		delay  = &testDelay{}
		robots = &testRobots{}

		testErr = errors.New("Test Error")
	)

	tests := []struct {
		Name   string
		Rules  *Rules
		Client Client
		Delay  Delay
		Robots RobotsTxt

		DelayWaitUsed  bool
		DelayStampUsed bool
		RobotsUsed     bool
		Err            error
	}{
		{"OK", &Rules{Delay: time.Second}, client, delay, robots, true, true, true, nil},
		{"clientIsNil", &Rules{}, nil /*Client*/, delay, robots, false, false, false, ErrClientIsNil},
		{"rulesIsNil", nil /*Rules*/, client, delay, robots, false, false, false, ErrRulesIsNil},

		{"noDelay", &Rules{}, client, nil /*Delay*/, robots, false, false, true, nil},
		{"noDelayStart", &Rules{}, client, delay, robots, false, true, true, nil},
		{"noRobots", &Rules{Delay: time.Second}, client, delay, nil /*Robots*/, true, true, false, nil},
		{"noDelayNoRobots", &Rules{}, client, nil /*Delay*/, nil /*Robots*/, false, false, false, nil},

		{
			"doErr",
			&Rules{Extra: map[string]any{"doErr": testErr}},
			client,
			delay,
			robots,
			false,
			true,
			false,
			testErr,
		},
		{
			"robotsErr",
			&Rules{Extra: map[string]any{"robotsErr": testErr}},
			client,
			nil, /*Delay*/
			robots,
			false,
			false,
			true,
			testErr,
		},
		{
			"doPanic",
			&Rules{Extra: map[string]any{"doPanic": testErr}},
			client,
			nil, /*Delay*/
			nil, /*Robots*/
			false,
			false,
			false,
			testErr,
		},
		{
			"robotsPanic",
			&Rules{Extra: map[string]any{"robotsPanic": testErr}},
			client,
			nil, /*Delay*/
			robots,
			false,
			false,
			true,
			testErr,
		},
	}

	for _, tt := range tests {
		c.Client = tt.Client
		c.Delay = tt.Delay
		c.RobotsTxt = tt.Robots

		t.Run(tt.Name, func(t *testing.T) {
			defer c.Clear()

			_, err := c.Do(tt.Rules)
			if (err != nil) && (tt.Err != nil) {
				if err.Error() != tt.Err.Error() {
					t.Fatal(err)
				}
				return

			} else if (err == nil) && (tt.Err == nil) {
				if delay.WaitUsed != tt.DelayWaitUsed {
					t.Fatal("Delay Wait")
				}

				if delay.DoneUsed != tt.DelayWaitUsed {
					t.Fatal("Delay Done")
				}

				if delay.StampUsed != tt.DelayStampUsed {
					t.Fatal("Delay Stamp")
				}

				if robots.IsAllowedUsed != tt.RobotsUsed {
					t.Fatal("RobotsTxt IsAllowed")
				}

				return
			}

			t.Fatal(err)
		})
	}
}

func TestExtract(t *testing.T) {
	var (
		c      = New()
		client = &testClient{}
		parser = &testParser{}

		testErr = errors.New("Test Error")

		wantOut = map[string]any{
			"response": map[string]any{"url": "http://example.com"},
			"data":     map[string]any{"title": "test"},
		}
	)
	c.RobotsTxt = &testRobots{}

	tests := []struct {
		Name      string
		Rules     *Rules
		Client    Client
		Parser    Parser
		ParseUsed bool
		Err       error
	}{
		{"OK", &Rules{Selectors: []*Selector{testSelector}}, client, parser, true, nil},

		{"ClientIsNil", &Rules{}, nil, parser, false, ErrClientIsNil},
		{"ParserIsNil", &Rules{}, client, nil, false, ErrParserIsNil},
		{"ParserIsNil2", &Rules{}, nil, nil, false, ErrParserIsNil},

		{
			"doErr",
			&Rules{
				Extra: map[string]any{"doErr": testErr},
			},
			client,
			parser,
			true,
			testErr,
		},
		{
			"robotsErr",
			&Rules{
				Extra: map[string]any{"robotsErr": testErr},
			},
			client,
			parser,
			true,
			testErr,
		},
		{
			"parserErr",
			&Rules{
				Selectors: []*Selector{testSelector},
				Extra:     map[string]any{"parserErr": testErr},
			},
			client,
			parser,
			true,
			testErr,
		},
		{
			"panic",
			&Rules{
				Selectors: []*Selector{testSelector},
				Extra:     map[string]any{"parserPanic": testErr},
			},
			client,
			parser,
			true,
			testErr,
		},
	}

	for _, tt := range tests {
		c.Client = tt.Client
		c.Parser = tt.Parser

		t.Run(tt.Name, func(t *testing.T) {
			defer c.Clear()

			output, err := c.Extract(tt.Rules)
			if (err != nil) && (tt.Err != nil) {
				if err.Error() != tt.Err.Error() {
					t.Fatal(err)
				}
				return

			} else if (err == nil) && (tt.Err == nil) {
				if tt.ParseUsed != parser.ParseUsed {
					t.Fatal("Parser Parse")
				}

				if !reflect.DeepEqual(output.Serializable(), wantOut) {
					t.Fatal("output Serializable")
				}
				return
			}

			t.Fatal(err)
		})
	}
}

func TestUserAgent(t *testing.T) {
	c := New()
	c.Client = &testClient{}

	tests := []struct {
		UserAgent, WantUserAgent string
	}{
		{"", DefaultUserAgent},
		{"  ", "  "},
		{"test/0.0.1", "test/0.0.1"},
	}

	rules := &Rules{Header: http.Header{}}
	for _, tt := range tests {
		name := "(" + tt.UserAgent + "_" + tt.WantUserAgent + ")"
		rules.Header.Set("User-Agent", tt.UserAgent)

		t.Run(name, func(t *testing.T) {
			_, err := c.Do(rules)
			if err != nil {
				t.Fatal(err)
			}

			if rules.Header.Get("User-Agent") != tt.WantUserAgent {
				t.Fatal("not equal")
			}
		})
	}
}

func TestClone(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		if !reflect.DeepEqual(testRules.Clone(), testRules) {
			t.Fatal("not Equal")
		}
	})

	t.Run("empty", func(t *testing.T) {
		rules := &Rules{Extra: make(map[string]any)}
		if !reflect.DeepEqual(rules.Clone(), rules) {
			t.Fatal("not Equal")
		}
	})
}

func TestSelectorRules(t *testing.T) {
	tests := []struct {
		SRC      *Rules
		Selector *Selector
		Rules    *Rules
	}{
		{testRules, testSelector, &Rules{
			Method:          testSelector.Method,
			Proxy:           testRules.Proxy,
			Header:          http.Header{"User-Agent": {"test/0.2.0"}},
			Timeout:         testRules.Timeout,
			Cookies:         testRules.Cookies,
			IgnoreRobotsTxt: testRules.IgnoreRobotsTxt,
			Delay:           testRules.Delay,
			Redirects:       testRules.Redirects,
			Selectors:       testSelector.Selectors,
			Extra:           testSelector.Extra,
		}},

		{
			&Rules{
				Method:  "POST",
				Proxy:   mustNewURL("http://proxy.example.com:8081"),
				Timeout: 5 * time.Second,
			},
			&Selector{
				Method:  "GET",
				Proxy:   mustNewURL("http://proxy.example.com:8080"),
				Header:  http.Header{"Accept": []string{"text/html"}},
				Timeout: 10 * time.Millisecond,
				Extra: map[string]any{
					"required": true,
				},
			},
			&Rules{
				Method:  "GET",
				Proxy:   mustNewURL("http://proxy.example.com:8080"),
				Header:  http.Header{"Accept": []string{"text/html"}},
				Timeout: 10 * time.Millisecond,
				Extra: map[string]any{
					"required": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			rules := tt.Selector.Rules(tt.SRC)

			if !reflect.DeepEqual(rules, tt.Rules) {
				t.Fatal("not equal")
			}
		})
	}
}

func TestRulesUnmarshalJSON(t *testing.T) {
	tests := []struct {
		Name     string
		RawRules []byte
		Rules    *Rules
		AnErr    bool
	}{
		{"OK", testRawRulesJSON, testRules, false},

		{
			"empty",
			[]byte(`{
				"Selectors": {
					"head": null,
					"body": {
						"selectors": null
					}
				}
			}`),
			&Rules{
				Selectors: []*Selector{
					{
						Name:  "body",
						Extra: make(map[string]any),
					},
				},
				Extra: make(map[string]any),
			},
			false,
		},

		{"nil", []byte(`{}`), &Rules{Extra: make(map[string]any)}, false},

		{"null", []byte(`null`), &Rules{}, false},

		{"fail", []byte(`"string"`), nil, true},

		{"badRules", testBadRawRulesJSON, nil, true},

		{"errInvalidSelectors", testBadRawRulesJSON_ErrInvalidSelectors, nil, true},

		{"errInvalidSelector", testBadRawRulesJSON_ErrInvalidSelector, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			newRules := &Rules{}
			defer ReleaseRules(newRules)

			err := json.Unmarshal(tt.RawRules, newRules)
			if (err != nil && !tt.AnErr) || (err == nil && tt.AnErr) {
				t.Fatal(err)

			} else if (err == nil) && !tt.AnErr {
				if !reflect.DeepEqual(newRules, tt.Rules) {
					t.Fatal("not equal")
				}
			}
		})
	}
}

func TestUtilFuncs(t *testing.T) {
	t.Run("toInt", func(t *testing.T) {
		tests := []struct {
			Input  any
			Output int
			AnErr  bool
		}{
			{1, 1, false},
			{1000, 1000, false},
			{1.5, 1, false},

			{"str", 0, true},
			{nil, 0, true},
			{false, 0, true},
		}

		for _, tt := range tests {
			name := fmt.Sprint(tt.Input)

			t.Run(name, func(t *testing.T) {
				out, err := toInt(tt.Input)
				if (err != nil && !tt.AnErr) || (err == nil && tt.AnErr) {
					t.Fatal(err)

				} else if (err == nil) && !tt.AnErr {
					if !reflect.DeepEqual(out, tt.Output) {
						t.Fatal("not equal")
					}
				}
			})
		}
	})

	t.Run("toHeader", func(t *testing.T) {
		tests := []struct {
			Input  any
			Output http.Header
			AnErr  bool
		}{
			{map[string]any{"User-Agent": "test/0.2.0"}, http.Header{"User-Agent": {"test/0.2.0"}}, false},
			{nil, http.Header{}, false},

			{"str", nil, true},
			{map[string]any{"User-Agent": 2.0}, nil, true},
			{map[string]any{"User-Agent": []any{"test/0.2.0", 2.0}}, nil, true},
		}

		for _, tt := range tests {
			name := fmt.Sprint(tt.Input)

			t.Run(name, func(t *testing.T) {
				out, err := toHeader(tt.Input)
				if (err != nil && !tt.AnErr) || (err == nil && tt.AnErr) {
					t.Fatal(err)

				} else if (err == nil) && !tt.AnErr {
					if !reflect.DeepEqual(out, tt.Output) {
						t.Fatal("not equal")
					}
				}
			})
		}
	})

	t.Run("toDuration", func(t *testing.T) {
		tests := []struct {
			Input  any
			Output time.Duration
			AnErr  bool
		}{
			{1, 1 * time.Millisecond, false},
			{1000, 1 * time.Second, false},
			{1.5, 1500000 * time.Nanosecond, false},

			{"str", 0, true},
			{nil, 0, true},
		}

		for _, tt := range tests {
			name := fmt.Sprint(tt.Input)

			t.Run(name, func(t *testing.T) {
				out, err := toDuration(tt.Input)
				if (err != nil && !tt.AnErr) || (err == nil && tt.AnErr) {
					t.Fatal(err)

				} else if (err == nil) && !tt.AnErr {
					if !reflect.DeepEqual(out, tt.Output) {
						t.Fatal("not equal")
					}
				}
			})
		}
	})
}

func TestErr(t *testing.T) {
	var (
		err1 = errors.New("err1")
		err2 = errors.New("err2")
		err3 = errors.New("err3")

		errs error
	)

	t.Run("AddError", func(t *testing.T) {
		errs = AddError(errs, "key", nil)
		errs = AddError(errs, "", err1)

		if errs != nil {
			t.Fatal("must be nil")
		}

		errs = AddError(errs, "err1", err1)
		errs = AddError(errs, "err2", err2)

		if err, ok := errs.(*Errs).Get("err1"); !ok {
			t.Fatal("error must exist")
		} else if !errors.Is(err, err1) {
			t.Fatal("error must be err1")
		}

		if err, ok := errs.(*Errs).Get("err2"); !ok {
			t.Fatal("error must exist")
		} else if !errors.Is(err, err2) {
			t.Fatal("error must be err2")
		}

		errsTemp := AddError(err1, "err2", err2)

		if err, ok := errsTemp.(*Errs).Get("#"); !ok {
			t.Fatal("error must exist")
		} else if !errors.Is(err, err1) {
			t.Fatal("error must be err1")
		}

		if err, ok := errsTemp.(*Errs).Get("err2"); !ok {
			t.Fatal("error must exist")
		} else if !errors.Is(err, err2) {
			t.Fatal("error must be err2")
		}
	})

	t.Run("subErrs", func(t *testing.T) {
		errs = errs.(*Errs).Add("key", nil)
		errs = errs.(*Errs).Add("", err1)

		subErrs := AddError(nil, "err3", err3)

		errs = AddError(errs, "sub", subErrs)

		if err, ok := errs.(*Errs).Get("sub"); !ok {
			t.Fatal("error must exist")
		} else if err, _ := err.(*Errs).Get("err3"); !errors.Is(err, err3) {
			t.Fatal("error must be err3")
		}
	})

	t.Run("err#n", func(t *testing.T) {
		errs = AddError(errs, "err1", err1)

		if err, ok := errs.(*Errs).Get("err1#1"); !ok {
			t.Fatal("error must exist")
		} else if !errors.Is(err, err1) {
			t.Fatal("error must be err1")
		}
	})

	want := map[string]any{
		"err1":   err1.Error(),
		"err1#1": err1.Error(),
		"err2":   err2.Error(),
		"sub": map[string]any{
			"err3": err3.Error(),
		},
	}

	result := make(map[string]any)
	if err := json.Unmarshal([]byte(errs.Error()), &result); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(result, want) {
		t.Fatal("not equal")
	}
}

func BenchmarkRulesJSON(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		var rules Rules
		if err := json.Unmarshal(testRawRulesJSON, &rules); err != nil {
			b.Fatal(err)
		}
		ReleaseRules(&rules)
	}
}

type testResp struct{}

func (resp *testResp) URL() *url.URL { return nil }

func (resp *testResp) StatusCode() int { return 0 }

func (resp *testResp) Header() http.Header { return nil }

func (resp *testResp) Body() io.ReadCloser { return nil }

func (resp *testResp) Redirects() []*url.URL { return nil }

func (resp *testResp) Serializable() map[string]any {
	return map[string]any{
		"url": "http://example.com",
	}
}

func (resp *testResp) Do(_ *Rules) (Response, error) {
	return resp, nil
}

func (resp *testResp) Extract(_ *Rules) (*Output, error) {
	return &Output{
		Response: resp,
		Data: map[string]any{
			"name": "testResp",
		},
	}, nil
}

type testClient struct {
	ClearUsed bool
}

func (c *testClient) Do(_ *Colibri, rules *Rules) (Response, error) {
	if err := rules.Extra["doErr"]; err != nil {
		return nil, err.(error)
	} else if v := rules.Extra["doPanic"]; v != nil {
		panic(v)
	}
	return &testResp{}, nil
}

func (c *testClient) Clear() { c.ClearUsed = true }

type testDelay struct {
	WaitUsed, DoneUsed, StampUsed, ClearUsed bool
}

func (d *testDelay) Wait(_ *url.URL, _ time.Duration) {
	d.ClearUsed = false
	d.WaitUsed = true
}

func (d *testDelay) Done(_ *url.URL) {
	d.ClearUsed = false
	d.DoneUsed = true
}

func (d *testDelay) Stamp(_ *url.URL) {
	d.ClearUsed = false
	d.StampUsed = true
}

func (d *testDelay) Clear() {
	d.ClearUsed = true
	d.WaitUsed = false
	d.DoneUsed = false
	d.StampUsed = false
}

type testRobots struct {
	IsAllowedUsed, ClearUsed bool
}

func (r *testRobots) IsAllowed(_ *Colibri, rules *Rules) error {
	r.IsAllowedUsed = true

	if err := rules.Extra["robotsErr"]; err != nil {
		return err.(error)
	} else if v := rules.Extra["robotsPanic"]; v != nil {
		panic(v)
	}
	return nil
}

func (r *testRobots) Clear() {
	r.ClearUsed = true
	r.IsAllowedUsed = false
}

type testParser struct {
	ParseUsed, ClearUsed bool
}

func (p *testParser) Match(_ string) bool { return true }

func (p *testParser) Parse(rules *Rules, _ Response) (map[string]any, error) {
	p.ParseUsed = true

	if err := rules.Extra["parserErr"]; err != nil {
		return nil, err.(error)
	} else if v := rules.Extra["parserPanic"]; v != nil {
		panic(v)
	}
	return map[string]any{"title": "test"}, nil
}

func (p *testParser) Clear() {
	p.ClearUsed = true
	p.ParseUsed = false
}
