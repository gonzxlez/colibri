package colibri

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestUtil_toInt(t *testing.T) {
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
		var (
			tt   = tt
			name = fmt.Sprint(tt.Input)
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

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
}

func TestUtil_toHeader(t *testing.T) {
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
		var (
			tt   = tt
			name = fmt.Sprint(tt.Input)
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

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
}

func TestUtil_toDuration(t *testing.T) {
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
		var (
			tt   = tt
			name = fmt.Sprint(tt.Input)
		)

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
}
