package util

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type rawQuery struct {
	sort   []string
	fields []string
	filter map[string][]string
	page   map[string]int
}

// parseQuery reads a raw querystring, or a URL carrying one. It splits on "&", "=", and ","
// before decoding, so an encoded delimiter stays part of its value.
func parseQuery(query string) (rawQuery, error) {
	parsed := rawQuery{filter: map[string][]string{}, page: map[string]int{}}

	// A "?" ends a URL prefix only if no parameter came before it; later it is data.
	if prefix, rest, found := strings.Cut(query, "?"); found && !strings.ContainsAny(prefix, "=&") {
		query = rest
	}

	for _, pair := range strings.Split(query, "&") {
		rawKey, rawValue, _ := strings.Cut(pair, "=")
		key, err := url.QueryUnescape(rawKey)
		if err != nil {
			return rawQuery{}, fmt.Errorf("invalid query parameter name: %w", err)
		}
		if err := parsed.add(key, rawValue); err != nil {
			return rawQuery{}, err
		}
	}
	return parsed, nil
}

func (q *rawQuery) add(key, rawValue string) error {
	name, inner, bracketed := strings.Cut(key, "[")
	if bracketed {
		var closed bool
		if inner, closed = strings.CutSuffix(inner, "]"); !closed {
			return nil
		}
	}

	switch {
	case !bracketed && (name == "sort" || name == "fields"):
		if rawValue == "" {
			return nil
		}
		values, err := splitList(rawValue)
		if err != nil {
			return fmt.Errorf("invalid %s value: %w", name, err)
		}
		if name == "sort" {
			q.sort = append(q.sort, values...)
		} else {
			q.fields = append(q.fields, values...)
		}
	case bracketed && name == "filter":
		values, err := splitList(rawValue)
		if err != nil {
			return fmt.Errorf("invalid filter value: %w", err)
		}
		q.filter[inner] = values
	case bracketed && name == "page":
		value, err := url.QueryUnescape(rawValue)
		if err != nil {
			return fmt.Errorf("invalid page value: %w", err)
		}
		number, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("page[%s] must be an integer", inner)
		}
		q.page[inner] = number
	}
	return nil
}

// splitList splits on literal commas, trimming the space around each, then decodes each item.
func splitList(raw string) ([]string, error) {
	items := strings.Split(raw, ",")
	for i, item := range items {
		value, err := url.QueryUnescape(item)
		if err != nil {
			return nil, err
		}
		if len(items) > 1 {
			value = strings.TrimSpace(value)
		}
		items[i] = value
	}
	return items, nil
}
