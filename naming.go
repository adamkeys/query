package query

import (
	"reflect"
	"regexp"
	"strings"
)

// elementInfo provides information about the element that is the source of the name.
type elementInfo interface {
	// Name returns a name provided by the element.
	Name() string
}

// defaultNamer identifies the namer used by default when inferring query names from struct identifiers.
var defaultNamer = standardNamer{}

// standardNamer implements a namer using conventions that query considers to be the default.
type standardNamer struct{}

// Ident returns "id".
func (s standardNamer) Ident(info elementInfo) string {
	return "id"
}

// Table returns the underscored element name.
func (s standardNamer) Table(info elementInfo) string {
	return underscore(info.Name())
}

// Column returns the underscored element name.
func (s standardNamer) Column(info elementInfo) string {
	return underscore(info.Name())
}

var (
	matchFirst     = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchRemaining = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// underscore returns the supplied string converted to snake case.
func underscore(str string) string {
	underscored := matchFirst.ReplaceAllString(str, "${1}_${2}")
	underscored = matchRemaining.ReplaceAllString(underscored, "${1}_${2}")
	return strings.ToLower(underscored)
}

// fieldInfo wraps a StructField to implement the [ElementInfo] interface.
type fieldInfo struct{ reflect.StructField }

func (n fieldInfo) Name() string { return n.StructField.Name }
