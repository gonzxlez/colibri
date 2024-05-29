package colibri

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestFindSelectors(t *testing.T) {
	var testErr = errors.New("test err")

	c := New()
	c.Client = &testClient{}
	c.Parser = &testParser{}

	tests := []struct {
		Name string

		Rules  *Rules
		Resp   Response
		Parent Node

		Output map[string]any
		ErrMap map[string]any
	}{
		{
			Name:   "resp_nil",
			Rules:  &Rules{},
			Resp:   nil,
			Parent: &testNode{},
			Output: nil,
			ErrMap: nil,
		},
		{
			Name:   "parent_nil",
			Rules:  &Rules{},
			Resp:   &testResponse{},
			Parent: nil,
			Output: nil,
			ErrMap: nil,
		},
		{
			Name: "OK",
			Rules: &Rules{Selectors: []*Selector{
				{Name: "title", Expr: "//title"},
				{Name: "empty", Expr: "!empty"},
				{
					Name: "body",
					Expr: "//body",
					Selectors: []*Selector{
						{Name: "urls", Expr: "//a/@href", All: true},
						{
							Name: "imgs",
							Expr: "//img",
							All:  true,
							Selectors: []*Selector{
								{Name: "src", Expr: "/@src"},
								{Name: "alt", Expr: "/@alt"},
							},
						},
					},
				},
			}},
			Resp:   &testResponse{},
			Parent: &testNode{},
			Output: map[string]any{
				"title": "test",
				"empty": nil,
				"body": map[string]any{
					"urls": []any{"test"},
					"imgs": []any{
						map[string]any{
							"src": "test",
							"alt": "test",
						},
					},
				},
			},
			ErrMap: nil,
		},
		{
			Name: "bad",
			Rules: &Rules{Selectors: []*Selector{
				{Name: "title", Expr: "!error"},
				{Name: "empty", Expr: "!empty"},
				{
					Name: "body",
					Expr: "//body",
					Selectors: []*Selector{
						{Name: "urls", Expr: "!error", All: true},
						{
							Name: "imgs",
							Expr: "//img",
							All:  true,
							Selectors: []*Selector{
								{Name: "src", Expr: "!error"},
								{Name: "alt", Expr: "/@alt"},
							},
						},
					},
				},
			}},
			Resp:   &testResponse{},
			Parent: &testNode{},
			Output: nil,
			ErrMap: map[string]any{
				"title": "test err",
				"body": map[string]any{
					"urls": "test err",
					"imgs": map[string]any{
						"0": map[string]any{
							"src": "test err",
						},
					},
				},
			},
		},
		{
			Name: "Follow",
			Rules: &Rules{Selectors: []*Selector{
				{
					Name:   "first",
					Expr:   "//a/@href", // test -> u.IsAbs
					Follow: true,
					Selectors: []*Selector{
						{Name: "title", Expr: "//title"},
					},
				},
				{
					Name:   "all",
					Expr:   "!link",
					All:    true,
					Follow: true,
					Selectors: []*Selector{
						{Name: "title", Expr: "//title"},
					},
				},
			}},
			Resp:   &testResponse{c: c},
			Parent: &testNode{},
			Output: map[string]any{
				"first": []any{
					map[string]any{
						"response": map[string]any{
							"url": "http://example.com",
						},
						"data": map[string]any{
							"title": "test",
						},
					},
				},
				"all": []any{
					map[string]any{
						"response": map[string]any{
							"url": "http://example.com",
						},
						"data": map[string]any{
							"title": "test",
						},
					},
				},
			},
			ErrMap: nil,
		},
		{
			Name: "Follow_bad",
			Rules: &Rules{Selectors: []*Selector{
				{
					Name:   "first",
					Expr:   "!number",
					Follow: true,
					Selectors: []*Selector{
						{Name: "title", Expr: "//title"},
					},
				},
				{
					Name:   "all",
					Expr:   "!link",
					All:    true,
					Follow: true,
					Extra:  map[string]any{"doErr": testErr},
					Selectors: []*Selector{
						{
							Name: "title",
							Expr: "//title",
						},
					},
				},
			}},
			Resp:   &testResponse{c: c},
			Parent: &testNode{},
			Output: nil,
			ErrMap: map[string]any{
				"first": map[string]any{
					"505": ErrMustBeString.Error(),
				},
				"all": map[string]any{
					"http://example.com/test": "test err",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			output, err := FindSelectors(tt.Rules, tt.Resp, tt.Parent)

			if (err != nil) && (tt.ErrMap != nil) {
				wantErr, _ := json.Marshal(tt.ErrMap)
				jsonErrs, _ := json.Marshal(err)

				if !reflect.DeepEqual(wantErr, jsonErrs) {
					t.Fatal(err)
				}
				return

			} else if (err == nil) && (tt.ErrMap == nil) {
				if !reflect.DeepEqual(output, tt.Output) {
					t.Fatal("not equal")
				}
				return
			}

			t.Fatal(err)
		})
	}
}
