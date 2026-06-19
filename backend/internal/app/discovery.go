package app

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var errDiscoveryQueryTooLong = errors.New("search query is too long")

// discoveryPolicy keeps alpha Event Discovery separate from the v1 Public Event Page.
type discoveryPolicy struct{}

func newDiscoveryPolicy() discoveryPolicy {
	return discoveryPolicy{}
}

func (discoveryPolicy) enabled() bool {
	return true
}

func (discoveryPolicy) normalizeSearchQuery(raw string) (string, error) {
	query := strings.ToLower(strings.TrimSpace(raw))
	if utf8.RuneCountInString(query) > 120 {
		return "", errDiscoveryQueryTooLong
	}

	return query, nil
}
