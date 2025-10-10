// Package assert provides testing helpers.
package assert

import (
	"slices"
	"strings"
	"testing"

	g "alin.ovh/gomponents"
)

// Equal checks for equality between the given expected string and the rendered Node string.
func Equal(t *testing.T, expected string, actual g.Node) {
	t.Helper()

	var b strings.Builder
	err := actual.Render(&b)
	if err != nil {
		t.Fatal("error rendering actual:", err)
	}
	if expected != b.String() {
		t.Fatalf(`expected "%v" but got "%v"`, expected, b.String())
	}
}

// OneOf checks if the given expected list includes the rendered Node string.
func OneOf(t *testing.T, expected []string, actual g.Node) {
	t.Helper()

	var b strings.Builder
	err := actual.Render(&b)
	if err != nil {
		t.Fatal("error rendering actual:", err)
	}
	if slices.Contains(expected, b.String()) {
		return
	}
	t.Fatalf(`expected one of "%v" but got "%v"`, expected, b.String())
}

// Error checks for a non-nil error.
func Error(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error is nil")
	}
}
