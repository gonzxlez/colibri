package colibri

import (
	"encoding/json"
	"net/http"
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
	"responseBodySize": 5000,
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
	"Method":           0.1,
	"url":              "http://example.com",
	"proxy":            "$$$$<:>8080",
	"header":           {"User-Agent": 0.2},
	"timeout":          "2.5",
	"cookies":          "true",
	"ignoreRobotsTXT":  1,
	"delay":            {},
	"responseBodySize": "5mb",
	"redirects":        true,
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

func TestRules_Clone(t *testing.T) {
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

func TestRules_UnmarshalJSON(t *testing.T) {
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
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

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

func TestSelector_Rules(t *testing.T) {
	tests := []struct {
		SRC      *Rules
		Selector *Selector
		Rules    *Rules
	}{
		{testRules, testSelector, &Rules{
			Method:           testSelector.Method,
			Proxy:            testRules.Proxy,
			Header:           http.Header{"User-Agent": {"test/0.2.0"}},
			Timeout:          testRules.Timeout,
			Cookies:          testRules.Cookies,
			IgnoreRobotsTxt:  testRules.IgnoreRobotsTxt,
			Delay:            testRules.Delay,
			Redirects:        testRules.Redirects,
			ResponseBodySize: testRules.ResponseBodySize,
			Selectors:        testSelector.Selectors,
			Extra:            testSelector.Extra,
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
		tt := tt

		t.Run("", func(t *testing.T) {
			t.Parallel()

			rules := tt.Selector.Rules(tt.SRC)
			if !reflect.DeepEqual(rules, tt.Rules) {
				t.Fatal("not equal")
			}
		})
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
