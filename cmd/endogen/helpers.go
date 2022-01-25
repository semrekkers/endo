package main

import (
	"fmt"
	"strconv"
	"strings"
)

// filterPrimary filters all primary key fields.
func filterPrimary(v []*field) []*field {
	fields, n := make([]*field, len(v)), 0
	for _, field := range v {
		if !field.PrimaryKey {
			fields[n] = field
			n++
		}
	}
	return fields[:n]
}

// toColumns returns a list of column names of fields.
func toColumns(fields []*field) []string {
	columns := make([]string, len(fields))
	for i := range fields {
		columns[i] = fields[i].Column
	}
	return columns
}

// joinStrings joins the given strings with a separator.
func joinStrings(sep string, a []string) string {
	return strings.Join(a, sep)
}

// mapToParams returns a list of placed parameters based on a.
func mapToParams(a []string) []string {
	v := make([]string, len(a))
	for i := range a {
		v[i] = "$" + strconv.Itoa(i+1)
	}
	return v
}

// lastArg returns the last argument count of a.
func lastArg(a []string) int {
	return len(a) + 1
}

// toFieldUpdates maps a to "<fieldName> = $<placedParameter>".
func toFieldUpdates(a []string) []string {
	v := make([]string, len(a))
	for i, field := range a {
		v[i] = fmt.Sprintf("%s = $%d", field, i+1)
	}
	return v
}
