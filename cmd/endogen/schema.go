package main

import (
	"fmt"
	"strings"
)

type Schema struct {
	Package    string        `yaml:"package"`
	Import     []string      `yaml:"import,omitempty"`
	ImportRepo []string      `yaml:"import_repo,omitempty"`
	NoRepoDecl bool          `yaml:"no_repo_decl"`
	Schema     []*Definition `yaml:"schema"`
}

type Definition struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"`
	Description    string   `yaml:"description,omitempty"`
	Plural         *string  `yaml:"plural,omitempty"`
	Table          *string  `yaml:"table,omitempty"`
	PatchType      *string  `yaml:"patch_type,omitempty"`
	PrimaryKey     *string  `yaml:"primary_key,omitempty"`
	OrderBy        *string  `yaml:"order_by,omitempty"`
	WithCreatedAt  *string  `yaml:"with_created_at,omitempty"`
	WithUpdatedAt  *string  `yaml:"with_updated_at,omitempty"`
	NoCreate       bool     `yaml:"no_create,omitempty"`
	NoUpdate       bool     `yaml:"no_update,omitempty"`
	NoDelete       bool     `yaml:"no_delete,omitempty"`
	InvokeOnCreate string   `yaml:"invoke_on_create,omitempty"`
	InvokeOnRead   string   `yaml:"invoke_on_read,omitempty"`
	InvokeOnUpdate string   `yaml:"invoke_on_update,omitempty"`
	Fields         []*Field `yaml:"fields"`

	PrimaryKeyField *Field `yaml:"-"`
	CreatedAtField  *Field `yaml:"-"`
	UpdatedAtField  *Field `yaml:"-"`
}

func (d *Definition) FieldsNoPrimaryKey() []*Field {
	if d.PrimaryKey == nil {
		return d.Fields
	}
	fields, n := make([]*Field, len(d.Fields)), 0
	for _, field := range d.Fields {
		if field != d.PrimaryKeyField {
			fields[n] = field
			n++
		}
	}
	return fields[:n]
}

func (d *Definition) PatchFields() []*Field {
	fields, n := make([]*Field, len(d.Fields)), 0
	for _, field := range d.Fields {
		if field != d.PrimaryKeyField && field != d.CreatedAtField && field != d.UpdatedAtField {
			fields[n] = field
			n++
		}
	}
	return fields[:n]
}

func (d *Definition) Columns() []string {
	columns := make([]string, len(d.Fields))
	for i, field := range d.Fields {
		columns[i] = field.Column
	}
	return columns
}

type Field struct {
	Name     string  `yaml:"name"`
	Column   string  `yaml:"column,omitempty"`
	Type     string  `yaml:"type"`
	JSONTag  *string `yaml:"json,omitempty"`
	Nullable bool    `yaml:"nullable,omitempty"`
}

func setDefaults(schema *Schema) error {
	for i, d := range schema.Schema {
		if d.Name == "" {
			return fmt.Errorf("schema: nameless definition in schema at index %d", i)
		}
		if err := setDefaultsDefinition(d); err != nil {
			return err
		}
	}
	return nil
}

func setDefaultsDefinition(d *Definition) error {
	if len(d.Fields) < 1 {
		return fmt.Errorf("schema: at least one field must be specified in definition %s", d.Name)
	}
	fields := make(map[string]*Field, len(d.Fields))
	for i, field := range d.Fields {
		if field.Name == "" {
			return fmt.Errorf("schema: field %d of definition %s: name is empty", i, d.Name)
		}
		if field.Type == "" {
			return fmt.Errorf("schema: field %s of definition %s: type is empty", field.Name, d.Name)
		}
		if field.Column == "" {
			field.Column = field.Name
		}
		if field.JSONTag == nil {
			field.JSONTag = &field.Column
		}
		if field.Nullable {
			if *field.JSONTag != "-" {
				tag := *field.JSONTag + ",omitempty"
				field.JSONTag = &tag
			}
		}

		fields[field.Column] = field
	}

	if d.Type == "" {
		d.Type = d.Name
	}
	if d.Plural == nil {
		plural := d.Name + "s"
		d.Plural = &plural
	}
	if d.Table == nil {
		table := strings.ToLower(*d.Plural)
		d.Table = &table
	}
	if d.PatchType == nil {
		patchType := d.Name + "Patch"
		d.PatchType = &patchType
	}
	if _, ok := fields["id"]; d.PrimaryKey == nil && ok {
		n := "id"
		d.PrimaryKey = &n
	}
	if _, ok := fields["id"]; d.OrderBy == nil && ok {
		n := "id"
		d.OrderBy = &n
	}
	if _, ok := fields["created_at"]; d.WithCreatedAt == nil && ok {
		n := "created_at"
		d.WithCreatedAt = &n
	}
	if _, ok := fields["updated_at"]; d.WithUpdatedAt == nil && ok {
		n := "updated_at"
		d.WithUpdatedAt = &n
	}

	if d.PrimaryKey != nil && *d.PrimaryKey != "" {
		field, ok := fields[*d.PrimaryKey]
		if !ok {
			return fmt.Errorf("schema: at primary_key in definition %s: field %s doesn't exist", d.Name, *d.PrimaryKey)
		}
		d.PrimaryKeyField = field
	}
	if d.WithCreatedAt != nil && *d.WithCreatedAt != "" {
		field, ok := fields[*d.WithCreatedAt]
		if !ok {
			return fmt.Errorf("schema: at with_created_at in definition %s: field %s doesn't exist", d.Name, *d.WithCreatedAt)
		}
		d.CreatedAtField = field
	}
	if d.WithUpdatedAt != nil && *d.WithUpdatedAt != "" {
		field, ok := fields[*d.WithUpdatedAt]
		if !ok {
			return fmt.Errorf("schema: at with_updated_at in definition %s: field %s doesn't exist", d.Name, *d.WithUpdatedAt)
		}
		d.UpdatedAtField = field
	}

	return nil
}
