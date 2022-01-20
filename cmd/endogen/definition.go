package main

import (
	"go/ast"
	"go/printer"
	"reflect"
	"regexp"
	"strings"
)

type definition struct {
	Package       string
	ExtraImports  []string
	Store         string
	GenerateStore bool
	ReadOnly      bool
	Models        []*model
}

type model struct {
	Name, Type, Plural, Table, PatchType string
	PrimaryKey                           *field
	OrderBy                              string
	Fields                               []*field
}

type field struct {
	Name, Column, Type string
}

func (m *model) FieldsNoPrimaryKey() []*field {
	if m.PrimaryKey == nil {
		return m.Fields
	}
	fields, n := make([]*field, len(m.Fields)), 0
	for _, field := range m.Fields {
		if field != m.PrimaryKey {
			fields[n] = field
			n++
		}
	}
	return fields[:n]
}

func (d *definition) addFile(f *ast.File) error {
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			if err := d.addModel(typeSpec.Name.Name, genDecl.Doc.Text(), structType); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *definition) addModel(name, doc string, s *ast.StructType) error {
	m := model{
		Name:    name,
		Type:    name,
		Plural:  parseDocArgument(doc, "plural"),
		Table:   parseDocArgument(doc, "table"),
		OrderBy: parseDocArgument(doc, "order by"),
	}

	if m.Plural == "" {
		m.Plural = m.Name + "s"
	}
	if m.Table == "" {
		m.Table = strings.ToLower(m.Plural)
	}
	if m.PatchType == "" {
		m.PatchType = m.Name + "Patch"
	}

	for _, field := range s.Fields.List {
		c := field
		m.addFields(c)
	}

	if m.OrderBy == "" && m.PrimaryKey != nil {
		m.OrderBy = m.PrimaryKey.Column
	}

	d.Models = append(d.Models, &m)
	return nil
}

func (m *model) addFields(f *ast.Field) error {
	typeString := sprintNode(f.Type)

	var (
		column    string
		isPrimary bool
	)
	if f.Tag != nil {
		tag := reflect.StructTag(f.Tag.Value[1 : len(f.Tag.Value)-1]).Get("db")
		parts := strings.SplitN(tag, ",", 2)
		column = parts[0]
		if column == "-" {
			return nil
		}
		if len(parts) == 2 {
			switch parts[1] {
			case "primary":
				isPrimary = true
			}
		}
	}

	for _, name := range f.Names {
		spec := &field{
			Name:   name.Name,
			Column: column,
			Type:   typeString,
		}
		if spec.Column == "" {
			spec.Column = spec.Name
		}

		if isPrimary {
			m.PrimaryKey = spec
		}

		m.Fields = append(m.Fields, spec)
	}

	return nil
}

func parseDocArgument(doc, name string) string {
	re := regexp.MustCompile(name + `:\s*([\w_ ]+)`)
	if match := re.FindStringSubmatch(doc); len(match) == 2 {
		return match[1]
	}
	return ""
}

func sprintNode(n ast.Node) string {
	var b strings.Builder
	p := printer.Config{
		Mode: printer.RawFormat,
	}
	if err := p.Fprint(&b, fset, n); err != nil {
		panic(err)
	}
	return b.String()
}
