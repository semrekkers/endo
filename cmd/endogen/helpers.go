package main

import (
	"fmt"
	"strconv"
	"strings"
)

func toColumns(fields []*field) []string {
	columns := make([]string, len(fields))
	for i := range fields {
		columns[i] = fields[i].Column
	}
	return columns
}

func joinStrings(sep string, a []string) string {
	return strings.Join(a, sep)
}

func mapToParams(a []string) []string {
	v := make([]string, len(a))
	for i := range a {
		v[i] = "$" + strconv.Itoa(i+1)
	}
	return v
}

func lastArg(a []string) int {
	return len(a) + 1
}

func toFieldUpdates(a []string) []string {
	v := make([]string, len(a))
	for i, field := range a {
		v[i] = fmt.Sprintf("%s = $%d", field, i+1)
	}
	return v
}
