package colibri

import (
	"fmt"
	"net/url"
	"strconv"
)

type Node interface {
	// Find finds the first child node that matches the selector.
	Find(selector *Selector) (Node, error)

	// FindAll finds all child nodes that match the selector.
	FindAll(selector *Selector) ([]Node, error)

	// Value returns the value of the node.
	Value() any
}

func FindSelectors(rules *Rules, resp Response, parent Node) (map[string]any, error) {
	if (resp == nil) || (parent == nil) {
		return nil, nil
	}

	var (
		result = make(map[string]any)
		errs   error
	)
	for _, selector := range rules.Selectors {
		found, err := findSelector(rules, resp, selector, parent)
		if err != nil {
			errs = AddError(errs, selector.Name, err)
			continue
		}
		result[selector.Name] = found
	}
	return result, errs
}

func findSelector(src *Rules, resp Response, selector *Selector, parent Node) (any, error) {
	if selector.All {
		return findAllSelector(src, resp, selector, parent)
	}

	child, err := parent.Find(selector)
	if err != nil {
		return nil, err
	} else if child == nil {
		return nil, nil
	}

	if selector.Follow {
		rules := selector.Rules(src)
		defer ReleaseRules(rules)

		return followSelector(rules, resp, child.Value())
	}

	if len(selector.Selectors) > 0 {
		rules := selector.Rules(src)
		defer ReleaseRules(rules)

		return FindSelectors(rules, resp, child)
	}
	return child.Value(), nil
}

func findAllSelector(src *Rules, resp Response, selector *Selector, parent Node) ([]any, error) {
	children, err := parent.FindAll(selector)
	if err != nil {
		return nil, err
	}

	var (
		result []any
		errs   error
	)
	if !selector.Follow && (len(selector.Selectors) > 0) {
		rules := selector.Rules(src)
		defer ReleaseRules(rules)

		for i, child := range children {
			found, err := FindSelectors(rules, resp, child)
			if err != nil {
				errs = AddError(errs, strconv.Itoa(i), err)
				continue
			}
			result = append(result, found)
		}

		return result, errs
	}

	for _, child := range children {
		result = append(result, child.Value())
	}

	if selector.Follow {
		rules := selector.Rules(src)
		defer ReleaseRules(rules)

		return followSelector(rules, resp, result...)
	}
	return result, errs
}

func followSelector(rules *Rules, resp Response, rawURL ...any) ([]any, error) {
	var (
		urls []*url.URL
		errs error
	)

	for _, rawU := range rawURL {
		u, err := ToURL(rawU)
		if err != nil {
			errs = AddError(errs, fmt.Sprintf("%v", rawU), err)
			continue
		}

		if !u.IsAbs() {
			u = resp.URL().ResolveReference(u)
		}
		urls = append(urls, u)
	}

	if errs != nil {
		return nil, errs
	}

	var result []any
	for _, u := range urls {
		cRules := rules.Clone()
		cRules.URL = u

		out, err := resp.Extract(cRules)
		if err != nil {
			errs = AddError(errs, u.String(), err)
			continue
		}

		result = append(result, out.Serializable())
		ReleaseRules(cRules)
	}

	return result, errs
}
