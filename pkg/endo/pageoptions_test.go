package endo_test

import (
	"fmt"
	"testing"

	"github.com/semrekkers/endo/pkg/endo"

	"github.com/stretchr/testify/assert"
)

func TestPageOptions(t *testing.T) {
	cases := []struct {
		Page, PerPage int
		Limit, Offset int
	}{
		{1, 100, 100, 0},
		{3, 100, 100, 200},
		{0, 0, 1, 0},
		{3, 0, 1, 2},
	}

	for _, test := range cases {
		t.Run(fmt.Sprintf("TestPageOptions: %+v", &test), func(t *testing.T) {
			po := endo.PageOptions{Page: test.Page, PerPage: test.PerPage}
			limit, offset := po.Args()
			assert.Equal(t, test.Limit, limit, "Limit not equal")
			assert.Equal(t, test.Offset, offset, "Offset not equal")
		})
	}
}
