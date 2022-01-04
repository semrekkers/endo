package main

import (
	"strconv"
	"strings"
)

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
