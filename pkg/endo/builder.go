package endo

import (
	"fmt"
	"strconv"
	"strings"
)

// A Builder is used to build a query string using Write methods. The zero value is ready to use. Do not copy a
// non-zero Builder, use Copy() instead.
type Builder struct {
	// FormatParam formats a parameter with index i.
	// If FormatParam is nil, Builder uses endo.FixedParam.
	FormatParam func(b *Builder, i int)

	s    strings.Builder
	args []interface{}
}

// Write appends s to the Builder's buffer. Returns the receiver Builder.
func (b *Builder) Write(s string) *Builder {
	b.s.WriteString(s)
	return b
}

// Writef formats according to a format specifier and appends to the Builder's buffer.
// Returns the receiver Builder.
func (b *Builder) Writef(format string, a ...interface{}) *Builder {
	fmt.Fprintf(&b.s, format, a...)
	return b
}

// WriteTrim appends the space trimmed s to the Builder's buffer. Returns the receiver Builder.
func (b *Builder) WriteTrim(s string) *Builder {
	b.s.WriteString(strings.TrimSpace(s))
	return b
}

// WriteWithArgs appends s with the arguments to the Builder's buffer. Returns the receiver Builder.
func (b *Builder) WriteWithArgs(s string, a ...interface{}) *Builder {
	b.s.WriteString(s)
	b.args = append(b.args, a...)
	return b
}

// WriteWithParams substitutes every parameter denoted by "{}" in s to a parameter formatted
// by Builder.FormatParam, and appends it along with the positioned argument from a, to
// the Builder's buffer. Returns the receiver Builder.
func (b *Builder) WriteWithParams(s string, p ...interface{}) *Builder {
	if b.FormatParam == nil {
		// Set default to FixedParam.
		b.FormatParam = FixedParam
	}
	b.s.Grow(len(s))
	for {
		i := strings.Index(s, "{}")
		if i == -1 {
			break
		}
		b.s.WriteString(s[:i])
		b.FormatParam(b, len(b.args))
		b.args = append(b.args, p[0])
		p, s = p[1:], s[i+2:] // advance
	}
	b.s.WriteString(s)
	return b
}

// KeyValue represents a key and value.
type KeyValue struct {
	Key   string
	Value interface{}
}

// Values represents multiple values. This can be helpful when you want to pass multiple
// values inside a KeyValue, for example.
type Values []interface{}

func (b *Builder) writeWithExpandedValue(s string, p interface{}) {
	if args, ok := p.(Values); ok {
		b.WriteWithParams(s, args...)
	} else {
		b.WriteWithParams(s, p)
	}
}

// WriteKeyValues formats every Key according to the format specifier, and
// substitutes every parameter denoted by "{}" to a parameter formatted by
// Builder.FormatParam, and appends it along with the positioned Value from a, to
// the Builder's buffer.
// Returns the receiver Builder.
func (b *Builder) WriteKeyValues(format string, sep string, a ...KeyValue) *Builder {
	if len(a) > 0 {
		b.writeWithExpandedValue(fmt.Sprintf(format, a[0].Key), a[0].Value)
		for _, arg := range a[1:] {
			b.s.WriteString(sep)
			b.writeWithExpandedValue(fmt.Sprintf(format, arg.Key), arg.Value)
		}
	}
	return b
}

// WithArgs appends a to the Builder's buffer. Returns the receiver Builder.
func (b *Builder) WithArgs(a ...interface{}) *Builder {
	b.args = append(b.args, a...)
	return b
}

// Copy returns a copy of the receiver Builder.
func (b *Builder) Copy() *Builder {
	c := &Builder{
		args: append([]interface{}(nil), b.args...),
	}
	c.s.WriteString(b.s.String())
	return c
}

// String returns the query string.
func (b *Builder) String() string {
	return b.s.String()
}

// Build returns the query string with it's arguments.
func (b *Builder) Build() (string, []interface{}) {
	return b.s.String(), b.args
}

// FixedParam writes a fixed parameter ($i) to the Builder.
func FixedParam(b *Builder, i int) {
	b.s.WriteByte('$')
	b.s.WriteString(strconv.Itoa(i + 1))
}

// QuestionMarkParam writes a question mark parameter (?) to the Builder.
func QuestionMarkParam(b *Builder, i int) {
	b.s.WriteByte('?')
}
