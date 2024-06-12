package colibri

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

var (
	// ErrInvalidHeader is returned when the value is an invalid header.
	ErrInvalidHeader = errors.New("invalid header")

	// ErrMustBeString is returned when the value is not a string.
	ErrMustBeString = errors.New("must be a string")

	// ErrMustBeNumber is returned when the value is not a number.
	ErrMustBeNumber = errors.New("must be a number")

	// ErrNotAssignable is returned when the value is not assignable to the field.
	ErrNotAssignable = errors.New("value is not assignable to field")
)

var (
	intType = reflect.TypeOf(int(0))

	urlType = reflect.TypeOf((*url.URL)(nil))

	headerType = reflect.TypeOf(http.Header{})

	durationType = reflect.TypeOf(time.Duration(0))

	selectorsType = reflect.TypeOf([]*Selector{})
)

func processRaw[T Rules | Selector](raw map[string]any, output *T) error {
	if raw == nil {
		return nil
	}

	var (
		rOutput = reflect.ValueOf(output).Elem()
		err     error
		errs    error
	)

	for key, value := range raw {
		field := rOutput.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(key, s)
		})

		if field.IsValid() && field.CanSet() {
			fieldType := field.Type()

			switch fieldType {
			case intType:
				value, err = toInt(value)
			case urlType:
				value, err = ToURL(value)
			case headerType:
				value, err = toHeader(value)
			case durationType:
				value, err = toDuration(value)
			case selectorsType:
				value, err = newSelectors(value)
			}

			if err != nil {
				errs = AddError(errs, key, err)
				continue
			}

			rValue := reflect.ValueOf(value)
			if !rValue.Type().AssignableTo(fieldType) {
				errs = AddError(errs, key, ErrNotAssignable)
				continue
			}

			field.Set(rValue)
			delete(raw, key)
		}
	}

	return errs
}

// ToURL converts a value to a *url.URL.
func ToURL(value any) (*url.URL, error) {
	rawURL, ok := value.(string)
	if ok {
		return url.Parse(rawURL)
	}
	return nil, ErrMustBeString
}

func toInt(value any) (int, error) {
	switch n := value.(type) {
	case int:
		return n, nil
	case float64:
		return int(n), nil
	}
	return 0, ErrMustBeNumber
}

func toHeader(value any) (http.Header, error) {
	header := http.Header{}

	if value == nil {
		return header, nil
	}

	headerMap, ok := value.(map[string]any)
	if !ok {
		return header, ErrInvalidHeader
	}

	for k, v := range headerMap {
		switch val := v.(type) {
		case string:
			header.Add(k, val)
		case []any:
			for _, e := range val {
				s, ok := e.(string)
				if !ok {
					return header, ErrInvalidHeader
				}
				header.Add(k, s)
			}

		default:
			return header, ErrInvalidHeader
		}
	}
	return header, nil
}

func toDuration(value any) (time.Duration, error) {
	switch d := value.(type) {
	case int:
		return time.Duration(d) * time.Millisecond, nil
	case float64:
		return time.Duration(d*1000000) * time.Nanosecond, nil
	}
	return 0, ErrMustBeNumber
}
