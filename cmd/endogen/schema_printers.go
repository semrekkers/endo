package main

import "strings"

func (d *Definition) ColumnsString() string {
	return strings.Join(d.Columns(), ", ")
}

func (d *Definition) ColumnsNoPrimaryKey() []string {
	fields := d.FieldsNoPrimaryKey()
	columns := make([]string, len(fields))
	for i, field := range fields {
		columns[i] = field.Column
	}
	return columns
}
