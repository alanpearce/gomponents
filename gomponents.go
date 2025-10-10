// Package gomponents provides HTML components in Go, that render to HTML 5.
//
// The primary interface is a [Node]. It defines a function Render, which should render the [Node]
// to the given writer as a string.
//
// All DOM elements and attributes can be created by using the [El] and [Attr] functions.
//
// The functions [Text], [Textf], [Raw], and [Rawf] can be used to create text nodes, either HTML-escaped or unescaped.
//
// See also helper functions [Map], [If], and [Iff] for mapping data to nodes and inserting them conditionally.
//
// There's also the [Group] type, which is a slice of [Node]-s that can be rendered as one [Node].
//
// For basic HTML elements and attributes, see the package html.
//
// For higher-level HTML components, see the package components.
//
// For HTTP helpers, see the package http.
package gomponents // import "alin.ovh/gomponents"

import (
	"fmt"
	"html/template"
	"io"
	"iter"
	"strings"
)

// Node is a DOM node that can Render itself to a [io.Writer].
type Node interface {
	Render(w io.Writer) error
}

// NodeWriter is a [Node] that can also be used as an [io.WriterTo]
type NodeWriter interface {
	Node
	io.WriterTo
}

// NodeType describes what type of [Node] it is, currently either an [ElementType] or an [AttributeType].
// This decides where a [Node] should be rendered.
// Nodes default to being [ElementType].
type NodeType int

const (
	ElementType = NodeType(iota)
	AttributeType
)

// nodeTypeDescriber can be implemented by Nodes to let callers know whether the [Node] is
// an [ElementType] or an [AttributeType].
// See [NodeType].
type nodeTypeDescriber interface {
	Type() NodeType
}

// NodeFunc is a render function that is also a [Node] of [ElementType].
type NodeFunc func(io.Writer) error

// Render satisfies [Node].
func (n NodeFunc) Render(w io.Writer) error {
	return n(w)
}

// Type satisfies [nodeTypeDescriber].
func (n NodeFunc) Type() NodeType {
	return ElementType
}

// String satisfies [fmt.Stringer].
func (n NodeFunc) String() string {
	var b strings.Builder
	_ = n.Render(&b)
	return b.String()
}

// NodeWriterFunc is a render function that is also a [Node] of [ElementType].
type NodeWriterFunc func(io.Writer) (int64, error)

// Render satisfies [Node].
func (n NodeWriterFunc) Render(w io.Writer) error {
	_, err := n(w)
	return err
}

// WriteTo satisfies [io.WriterTo].
func (n NodeWriterFunc) WriteTo(w io.Writer) (int64, error) {
	return n(w)
}

// Type satisfies [nodeTypeDescriber].
func (n NodeWriterFunc) Type() NodeType {
	return ElementType
}

// String satisfies [fmt.Stringer].
func (n NodeWriterFunc) String() string {
	var b strings.Builder
	_ = n.Render(&b)
	return b.String()
}

var (
	lt      = []byte("<")
	gt      = []byte(">")
	ltSlash = []byte("</")
)

// El creates an element DOM [Node] with a name and child Nodes.
// See https://dev.w3.org/html5/spec-LC/syntax.html#elements-0 for how elements are rendered.
// No tags are ever omitted from normal tags, even though it's allowed for elements given at
// https://dev.w3.org/html5/spec-LC/syntax.html#optional-tags
// If an element is a void element, non-attribute children nodes are ignored.
// Use this if no convenience creator exists in the html package.
func El(name string, children ...Node) Node {
	return NodeWriterFunc(func(w io.Writer) (total int64, err error) {
		var n int
		sw, ok := w.(io.StringWriter)

		n, err = w.Write(lt)
		if err != nil {
			return
		}
		total += int64(n)

		if ok {
			n, err = sw.WriteString(name)
			if err != nil {
				return
			}
		} else {
			n, err = w.Write([]byte(name))
			if err != nil {
				return
			}
		}
		total += int64(n)

		for _, c := range children {
			n, err := renderChild(w, c, AttributeType)
			if err != nil {
				return total, err
			}
			total += n
		}

		n, err = w.Write(gt)
		if err != nil {
			return
		}
		total += int64(n)

		if isVoidElement(name) {
			return
		}

		for _, c := range children {
			n, err := renderChild(w, c, ElementType)
			if err != nil {
				return total, err
			}
			total += n
		}

		n, err = w.Write(ltSlash)
		if err != nil {
			return
		}
		total += int64(n)

		if ok {
			n, err = sw.WriteString(name)
		} else {
			n, err = w.Write([]byte(name))
		}
		if err != nil {
			return
		}
		total += int64(n)

		n, err = w.Write(gt)
		if err != nil {
			return
		}
		total += int64(n)

		return
	})
}

// renderChild c to the given writer w if the node type is desiredType.
func renderChild(w io.Writer, c Node, desiredType NodeType) (int64, error) {
	var count int64
	if c == nil {
		return count, nil
	}

	// Rendering groups like this is still important even though a group can render itself,
	// since otherwise attributes will sometimes be ignored.
	if g, ok := c.(Group); ok {
		for _, groupC := range g {
			n, err := renderChild(w, groupC, desiredType)
			if err != nil {
				return count, err
			}
			count += n
		}
		return count, nil
	}

	switch desiredType {
	case ElementType:
		if p, ok := c.(nodeTypeDescriber); !ok || p.Type() == desiredType {
			n, err := c.(NodeWriter).WriteTo(w)
			if err != nil {
				return count, err
			}
			count += n
		}
	case AttributeType:
		if p, ok := c.(nodeTypeDescriber); ok && p.Type() == desiredType {
			n, err := c.(NodeWriter).WriteTo(w)
			if err != nil {
				return count, err
			}
			count += n
		}
	}

	return count, nil
}

// voidElements don't have end tags and must be treated differently in the rendering.
// See https://dev.w3.org/html5/spec-LC/syntax.html#void-elements
var voidElements = map[string]struct{}{
	"area":    {},
	"base":    {},
	"br":      {},
	"col":     {},
	"command": {},
	"embed":   {},
	"hr":      {},
	"img":     {},
	"input":   {},
	"keygen":  {},
	"link":    {},
	"meta":    {},
	"param":   {},
	"source":  {},
	"track":   {},
	"wbr":     {},
}

func isVoidElement(name string) bool {
	_, ok := voidElements[name]
	return ok
}

var (
	space      = []byte(" ")
	equalQuote = []byte(`="`)
	quote      = []byte(`"`)
)

// Attr creates an attribute DOM [Node] with a name and optional value.
// If only a name is passed, it's a name-only (boolean) attribute (like "required").
// If a name and value are passed, it's a name-value attribute (like `class="header"`).
// More than one value make [Attr] panic.
// Use this if no convenience creator exists in the html package.
func Attr(name string, value ...string) Node {
	if len(value) > 1 {
		panic("attribute must be just name or name and value pair")
	}

	return attrFunc(func(w io.Writer) (total int64, err error) {
		var n int
		sw, ok := w.(io.StringWriter)

		n, err = w.Write(space)
		if err != nil {
			return
		}
		total += int64(n)

		// Attribute name
		if ok {
			n, err = sw.WriteString(name)
		} else {
			n, err = w.Write([]byte(name))
		}
		if err != nil {
			return
		}
		total += int64(n)

		if len(value) == 0 {
			return
		}

		n, err = w.Write(equalQuote)
		if err != nil {
			return
		}
		total += int64(n)

		// Attribute value
		if ok {
			n, err = sw.WriteString(template.HTMLEscapeString(value[0]))
		} else {
			n, err = w.Write([]byte(template.HTMLEscapeString(value[0])))
		}
		if err != nil {
			return
		}
		total += int64(n)

		n, err = w.Write(quote)
		if err != nil {
			return
		}
		total += int64(n)

		return
	})
}

// attrFunc is a render function that is also a [Node] of [AttributeType].
// It's basically the same as [NodeWriterFunc], but for attributes.
type attrFunc func(io.Writer) (int64, error)

// Render satisfies [Node].
func (a attrFunc) Render(w io.Writer) error {
	_, err := a(w)
	return err
}

// WriteTo satisfies [io.WriterTo].
func (a attrFunc) WriteTo(w io.Writer) (int64, error) {
	return a(w)
}

// Type satisfies [nodeTypeDescriber].
func (a attrFunc) Type() NodeType {
	return AttributeType
}

// String satisfies [fmt.Stringer].
func (a attrFunc) String() string {
	var b strings.Builder
	_ = a.Render(&b)
	return b.String()
}

// Text creates a text DOM [Node] that Renders the escaped string t.
func Text(t string) Node {
	return NodeWriterFunc(func(w io.Writer) (int64, error) {
		if w, ok := w.(io.StringWriter); ok {
			n, err := w.WriteString(template.HTMLEscapeString(t))
			return int64(n), err
		}
		n, err := w.Write([]byte(template.HTMLEscapeString(t)))
		return int64(n), err
	})
}

// Textf creates a text DOM [Node] that Renders the interpolated and escaped string format.
func Textf(format string, a ...any) Node {
	return NodeWriterFunc(func(w io.Writer) (int64, error) {
		if w, ok := w.(io.StringWriter); ok {
			n, err := w.WriteString(template.HTMLEscapeString(fmt.Sprintf(format, a...)))
			return int64(n), err
		}
		n, err := w.Write([]byte(template.HTMLEscapeString(fmt.Sprintf(format, a...))))
		return int64(n), err
	})
}

// Raw creates a text DOM [Node] that just Renders the unescaped string t.
func Raw(t string) Node {
	return NodeWriterFunc(func(w io.Writer) (int64, error) {
		if w, ok := w.(io.StringWriter); ok {
			n, err := w.WriteString(t)
			return int64(n), err
		}
		n, err := w.Write([]byte(t))
		return int64(n), err
	})
}

// Rawf creates a text DOM [Node] that just Renders the interpolated and unescaped string format.
func Rawf(format string, a ...any) Node {
	return NodeWriterFunc(func(w io.Writer) (int64, error) {
		if w, ok := w.(io.StringWriter); ok {
			n, err := w.WriteString(fmt.Sprintf(format, a...))
			return int64(n), err
		}
		n, err := fmt.Fprintf(w, format, a...)
		return int64(n), err
	})
}

// Map a slice of anything to a [Group] (which is just a slice of [Node]-s).
func Map[T any](ts []T, cb func(T) Node) Group {
	nodes := make([]Node, 0, len(ts))
	for _, t := range ts {
		nodes = append(nodes, cb(t))
	}
	return nodes
}

// Map a slice of anything to a [Group] (which is just a slice of [Node]-s).
func MapWithIndex[T any](ts []T, cb func(int, T) Node) Group {
	nodes := make([]Node, 0, len(ts))
	for k, t := range ts {
		nodes = append(nodes, cb(k, t))
	}

	return nodes
}

// Map a map of anything to a [Group] (which is just a slice of [Node]-s).
func MapMap[K comparable, T any](ts map[K]T, cb func(K, T) Node) Group {
	nodes := make([]Node, 0, len(ts))
	for k, t := range ts {
		nodes = append(nodes, cb(k, t))
	}

	return nodes
}

// Map an iterator of anything to an IterNode (which is just an [iter.Seq] of [Node]-s)
func MapIter[T any](ts iter.Seq[T], cb func(T) Node) IterNode {
	return IterNode{
		func(yield func(Node) bool) {
			for t := range ts {
				if !yield(cb(t)) {
					return
				}
			}
		},
	}
}

// Map an iterator of pairs of values to an IterNode (which is just an [iter.Seq] of [Node]-s)
func MapIter2[K, V any](ts iter.Seq2[K, V], cb func(K, V) Node) IterNode {
	return IterNode{
		func(yield func(Node) bool) {
			for k, v := range ts {
				if !yield(cb(k, v)) {
					return
				}
			}
		},
	}
}

type IterNode struct {
	iter.Seq[Node]
}

func (it IterNode) WriteTo(w io.Writer) (int64, error) {
	var written int64
	for node := range it.Seq {
		n, err := node.(NodeWriter).WriteTo(w)
		if err != nil {
			return written, err
		}

		written += n
	}

	return written, nil
}

func (it IterNode) Render(w io.Writer) error {
	_, err := it.WriteTo(w)
	return err
}

// Group a slice of [Node]-s into one Node, while still being usable like a regular slice of [Node]-s.
// A [Group] can render directly, but if any of the direct children are [AttributeType], they will be ignored,
// to not produce invalid HTML.
type Group []Node

// String satisfies [fmt.Stringer].
func (g Group) String() string {
	var b strings.Builder
	_ = g.Render(&b)
	return b.String()
}

// WriteTo satisfies [io.Writer].
func (g Group) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for _, c := range g {
		n, err := renderChild(w, c, ElementType)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// Render satisfies [Node].
func (g Group) Render(w io.Writer) error {
	_, err := g.WriteTo(w)
	return err
}

// If condition is true, return the given [Node]. Otherwise, return nil.
// This helper function is good for inlining elements conditionally.
// If it's important that the given [Node] is only evaluated if condition is true
// (for example, when using nilable variables), use [Iff] instead.
func If(condition bool, n Node, otherwise ...Node) Node {
	var o Node
	switch len(otherwise) {
	case 0:
	case 1:
		o = otherwise[0]
	default:
		panic("If must have just one or two nodes")
	}
	if condition {
		return n
	}
	return o
}

// Iff condition is true, call the given function. Otherwise, return nil.
// This helper function is good for inlining elements conditionally when the node depends on nilable data,
// or some other code that could potentially panic.
// If you just need simple conditional rendering, see [If].
func Iff(condition bool, f func() Node, otherwise ...func() Node) Node {
	var o func() Node
	switch len(otherwise) {
	case 0:
	case 1:
		o = otherwise[0]
	default:
		panic("Iff must have just one or two nodes")
	}
	if condition {
		return f()
	}
	if o != nil {
		return o()
	}
	return nil
}
