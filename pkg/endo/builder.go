package endo

import (
	"fmt"
	"strconv"
	"strings"
)

// A Builder is used to build a query string using Write methods. The zero value is ready to use. Do not copy a
// non-zero Builder, use Copy() instead.
type Builder struct {
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

// WriteWithPlaced substitutes the parameter denoted by '?' in s to the correct positioned parameter ('$n'), and
// appends it along with argument p to the Builder's buffer. Returns the receiver Builder.
func (b *Builder) WriteWithPlaced(s string, p interface{}) *Builder {
	if i := strings.IndexByte(s, '?'); i != -1 {
		b.s.WriteString(s[:i])
		b.s.WriteString("$")
		b.s.WriteString(strconv.Itoa(len(b.args) + 1))
		s = s[i+1:]
	}
	b.s.WriteString(s)
	b.args = append(b.args, p)
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
