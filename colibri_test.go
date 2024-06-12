package colibri

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"
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
		Method:           "GET",
		URL:              mustNewURL("http://example.com"),
		Proxy:            mustNewURL("http://proxy.example.com:8080"),
		Header:           http.Header{"User-Agent": {"test/0.2.0"}},
		Timeout:          2500000 * time.Nanosecond,
		Cookies:          true,
		IgnoreRobotsTxt:  true,
		Delay:            1500000 * time.Nanosecond,
		Redirects:        3,
		ResponseBodySize: 5000,
		Selectors:        []*Selector{testSelector},
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
	var testErr = errors.New("test err")

	tests := []struct {
		Name   string
		Rules  *Rules
		Client bool
		Delay  bool
		Robots bool

		DelayWaitUsed  bool
		DelayStampUsed bool
		RobotsUsed     bool
		Err            error
	}{
		{
			Name:           "OK",
			Rules:          &Rules{Delay: time.Second},
			Client:         true,
			Delay:          true,
			Robots:         true,
			DelayWaitUsed:  true,
			DelayStampUsed: true,
			RobotsUsed:     true,
		},
		{
			Name:   "clientIsNil",
			Rules:  &Rules{},
			Delay:  true,
			Robots: true,
			Err:    ErrClientIsNil,
		},
		{
			Name:   "rulesIsNil",
			Client: true,
			Delay:  true,
			Robots: true,
			Err:    ErrRulesIsNil,
		},
		{
			Name:       "noDelay",
			Rules:      &Rules{},
			Client:     true,
			Robots:     true,
			RobotsUsed: true,
		},
		{
			Name:           "noDelayStart",
			Rules:          &Rules{Delay: -1},
			Client:         true,
			Delay:          true,
			Robots:         true,
			DelayStampUsed: true,
			RobotsUsed:     true,
		},
		{
			Name:           "noRobots",
			Rules:          &Rules{Delay: time.Second},
			Client:         true,
			Delay:          true,
			DelayWaitUsed:  true,
			DelayStampUsed: true,
		},
		{
			Name:   "noDelayNoRobots",
			Rules:  &Rules{},
			Client: true,
		},
		{
			Name:           "doErr",
			Rules:          &Rules{Extra: map[string]any{"doErr": testErr}},
			Client:         true,
			Delay:          true,
			Robots:         true,
			DelayStampUsed: true,
			Err:            testErr,
		},
		{
			Name:       "robotsErr",
			Rules:      &Rules{Extra: map[string]any{"robotsErr": testErr}},
			Client:     true,
			Robots:     true,
			RobotsUsed: true,
			Err:        testErr,
		},
		{
			Name:   "doPanic",
			Rules:  &Rules{Extra: map[string]any{"doPanic": testErr}},
			Client: true,
			Err:    testErr,
		},
		{
			Name:       "robotsPanic",
			Rules:      &Rules{Extra: map[string]any{"robotsPanic": testErr}},
			Client:     true,
			Robots:     true,
			RobotsUsed: true,
			Err:        testErr,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			var (
				c      = New()
				delay  = &testDelay{}
				robots = &testRobots{}
			)

			if tt.Client {
				c.Client = &testClient{}
			}

			if tt.Delay {
				c.Delay = delay
			}

			if tt.Robots {
				c.RobotsTxt = robots
			}

			_, err := c.Do(tt.Rules)
			if (err != nil) && (tt.Err != nil) {
				if err.Error() != tt.Err.Error() {
					t.Fatal(err)
				}
				return

			} else if (err == nil) && (tt.Err == nil) {
				if delay.WaitUsed != tt.DelayWaitUsed {
					t.Fatal("Delay.Wait =", delay.WaitUsed)
				}

				if delay.DoneUsed != tt.DelayWaitUsed {
					t.Fatal("Delay.Done =", delay.DoneUsed)
				}

				if delay.StampUsed != tt.DelayStampUsed {
					t.Fatal("Delay.Stamp =", delay.StampUsed)
				}

				if robots.IsAllowedUsed != tt.RobotsUsed {
					t.Fatal("RobotsTxt.IsAllowed =", robots.IsAllowedUsed)
				}

				return
			}

			t.Fatal(err)
		})
	}
}

func TestExtract(t *testing.T) {
	var (
		testErr = errors.New("test err")

		wantOut = map[string]any{
			"response": map[string]any{"url": "http://example.com"},
			"data":     map[string]any{"title": "test"},
		}
	)

	tests := []struct {
		Name      string
		Rules     *Rules
		Client    bool
		Parser    bool
		ParseUsed bool
		Err       error
	}{
		{
			Name: "OK",
			Rules: &Rules{Selectors: []*Selector{
				{Name: "title", Expr: "//title"},
			}},
			Client:    true,
			Parser:    true,
			ParseUsed: true,
		},
		{
			Name:   "ClientIsNil",
			Rules:  &Rules{},
			Parser: true,
			Err:    ErrClientIsNil,
		},
		{
			Name:   "ParserIsNil",
			Rules:  &Rules{},
			Client: true,
			Err:    ErrParserIsNil,
		},
		{
			Name:  "ParserIsNil2",
			Rules: &Rules{},
			Err:   ErrParserIsNil,
		},
		{
			Name: "doErr",
			Rules: &Rules{
				Extra: map[string]any{"doErr": testErr},
			},
			Client:    true,
			Parser:    true,
			ParseUsed: true,
			Err:       testErr,
		},
		{
			Name: "robotsErr",
			Rules: &Rules{
				Extra: map[string]any{"robotsErr": testErr},
			},
			Client:    true,
			Parser:    true,
			ParseUsed: true,
			Err:       testErr,
		},
		{
			Name: "parserErr",
			Rules: &Rules{
				Selectors: []*Selector{testSelector},
				Extra:     map[string]any{"parserErr": testErr},
			},
			Client:    true,
			Parser:    true,
			ParseUsed: true,
			Err:       testErr,
		},
		{
			Name: "panic",
			Rules: &Rules{
				Selectors: []*Selector{testSelector},
				Extra:     map[string]any{"parserPanic": testErr},
			},
			Client:    true,
			Parser:    true,
			ParseUsed: true,
			Err:       testErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var (
				c      = New()
				parser = &testParser{}
			)

			c.RobotsTxt = &testRobots{}

			if tt.Client {
				c.Client = &testClient{}
			}

			if tt.Parser {
				c.Parser = parser
			}

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
					t.Fatal("not equal")
				}
				return
			}

			t.Fatal(err)
		})
	}

	t.Run("Output.MarshalJSON", func(t *testing.T) {
		out := &Output{
			Response: &testResponse{},
			Data: map[string]any{
				"title": "test",
			},
		}

		data, err := out.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}

		resp := &testResponse{}
		want := map[string]any{
			"response": resp.Serializable(),
			"data": map[string]any{
				"title": "test",
			},
		}

		outMap := make(map[string]any)
		if err := json.Unmarshal(data, &outMap); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(outMap, want) {
			t.Fatal("not equal")
		}
	})
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

	for _, tt := range tests {
		var (
			tt    = tt
			name  = "(" + tt.UserAgent + "_" + tt.WantUserAgent + ")"
			rules = &Rules{Header: http.Header{}}
		)

		rules.Header.Set("User-Agent", tt.UserAgent)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

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

func TestClear(t *testing.T) {
	var (
		c      = New()
		client = &testClient{}
		delay  = &testDelay{}
		robots = &testRobots{}
		parser = &testParser{}
	)

	c.Clear()

	c.Client = client
	c.Delay = delay
	c.RobotsTxt = robots
	c.Parser = parser

	if client.ClearUsed || delay.ClearUsed || robots.ClearUsed || parser.ClearUsed {
		t.Fatal("clear used")
	}

	c.Clear()

	if !client.ClearUsed || !delay.ClearUsed || !robots.ClearUsed || !parser.ClearUsed {
		t.Fatal("clear used")
	}
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

type testResponse struct {
	c *Colibri
}

func (resp *testResponse) URL() *url.URL { return mustNewURL("http://example.com") }

func (resp *testResponse) StatusCode() int { return 200 }

func (resp *testResponse) Header() http.Header { return http.Header{} }

func (resp *testResponse) Body() io.ReadCloser { return nil }

func (resp *testResponse) Redirects() []*url.URL { return nil }

func (resp *testResponse) Serializable() map[string]any {
	return map[string]any{
		"url": resp.URL().String(),
	}
}

func (resp *testResponse) Do(rules *Rules) (Response, error) { return resp.c.Do(rules) }

func (resp *testResponse) Extract(rules *Rules) (*Output, error) { return resp.c.Extract(rules) }

type testClient struct {
	ClearUsed bool
}

func (client *testClient) Do(c *Colibri, rules *Rules) (Response, error) {
	if err := rules.Extra["doErr"]; err != nil {
		return nil, err.(error)
	} else if v := rules.Extra["doPanic"]; v != nil {
		panic(v)
	}

	return &testResponse{c: c}, nil
}

func (client *testClient) Clear() {
	client.ClearUsed = true
}

type testDelay struct {
	WaitUsed, DoneUsed, StampUsed bool
	ClearUsed                     bool
}

func (d *testDelay) Wait(_ *url.URL, _ time.Duration) { d.WaitUsed = true }

func (d *testDelay) Done(_ *url.URL) { d.DoneUsed = true }

func (d *testDelay) Stamp(_ *url.URL) { d.StampUsed = true }

func (d *testDelay) Clear() {
	d.ClearUsed = true
}

type testRobots struct {
	IsAllowedUsed bool
	ClearUsed     bool
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
}

type testParser struct {
	ParseUsed bool
	ClearUsed bool
}

func (p *testParser) Match(_ string) bool { return true }

func (p *testParser) Parse(rules *Rules, _ Response) (Node, error) {
	p.ParseUsed = true

	if err := rules.Extra["parserErr"]; err != nil {
		return nil, err.(error)
	} else if v := rules.Extra["parserPanic"]; v != nil {
		panic(v)
	}
	return &testNode{}, nil
}

func (p *testParser) Clear() {
	p.ClearUsed = true
}

type testNode struct {
	value any
}

func (node *testNode) Find(selector *Selector) (Node, error) {
	if selector.Expr == "!empty" {
		return nil, nil
	} else if selector.Expr == "!error" {
		return nil, errors.New("test err")
	} else if selector.Expr == "!number" {
		return &testNode{value: 505}, nil
	}
	return &testNode{}, nil
}

func (node *testNode) FindAll(selector *Selector) ([]Node, error) {
	if selector.Expr == "!error" {
		return nil, errors.New("test err")
	}
	return []Node{&testNode{}}, nil
}

func (node *testNode) Value() any {
	if node.value != nil {
		return node.value
	}
	return "test"
}
