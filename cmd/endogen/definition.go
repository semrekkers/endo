package main

import (
	"errors"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// definition represents a schema to generate code for.
type definition struct {
	Package             string
	ExtraImports        []string
	ModelsExternal      bool
	ModelsImportPath    string
	ModelsImportAlias   string
	ModelsPackageName   string
	ModelsPackagePrefix string
	Store               string
	GenerateStore       bool
	ReadOnly            bool
	Models              []*model
}

// model repressets a single model (table or view) to generate code for.
type model struct {
	Name       string // model name in source code
	Type       string // model type in source code
	ReadOnly   bool   // model is read-only
	Immutable  bool   // model is immutable (no updates)
	Plural     string // plural of name
	Table      string // table name in database
	PatchType  string // patch type name to be generated
	PrimaryKey *field // primary key field, if any
	OrderBy    string // order by clause to use for result set, if any

	fields []*field
}

type field struct {
	Name       string // field name in source code
	Column     string // column name in model
	Type       string // field type in source code
	PrimaryKey bool   // whether this field is a primary key
	Exclude    bool   // whether this field is excluded from the model
	ReadOnly   bool   // whether this field is read-only
}

// Fields returns the fields of the model. If forWrite is true, only
// writable fields are returned.
func (m *model) Fields(forWrite bool) []*field {
	fields, n := make([]*field, len(m.fields)), 0
	for _, field := range m.fields {
		if field.Exclude || (forWrite && field.ReadOnly) {
			continue
		}
		fields[n] = field
		n++
	}
	return fields[:n]
}

// addFile parses the given file and adds the models it defines to
// the definition.
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
			typeName := typeSpec.Name.Name
			if d.ModelsExternal && !ast.IsExported(typeName) {
				// Unexported type of external package, ignore because it isn't accessible.
				continue
			}

			if err := d.addModel(typeSpec.Name.Name, genDecl.Doc.Text(), structType); err != nil {
				return err
			}
		}
	}

	return nil
}

// addModel parses the given StructType as model.
func (d *definition) addModel(name, doc string, s *ast.StructType) error {
	if strings.Contains(doc, "endo-ignore") {
		return nil
	}
	m := model{
		Name:     name,
		Type:     name,
		ReadOnly: d.ReadOnly,
		Plural:   parseDocArgument(doc, "plural"),
		Table:    parseDocArgument(doc, "table"),
		OrderBy:  parseDocArgument(doc, "order by"),
	}

	if arg := parseDocArgument(doc, "read-only"); arg != "" {
		m.ReadOnly, _ = strconv.ParseBool(arg)
	}
	if arg := parseDocArgument(doc, "immutable"); arg != "" {
		m.Immutable, _ = strconv.ParseBool(arg)
	}
	if m.Plural == "" {
		// If no plural is specified, try a simple pluralization.
		m.Plural = m.Name + "s"
	}
	if m.Table == "" {
		// If no table is specified, derive it from the plural.
		m.Table = strings.ToLower(m.Plural)
	}
	if m.PatchType == "" {
		// If no patch type is specified, derive it from the name.
		m.PatchType = m.Name + "Patch"
	}

	for _, field := range s.Fields.List {
		c := field
		m.addFields(d, c)
	}

	if m.OrderBy == "" && m.PrimaryKey != nil {
		// If a primary key is present, use it as the default order by, unless otherwise specified.
		m.OrderBy = m.PrimaryKey.Column
	}

	d.Models = append(d.Models, &m)
	return nil
}

func (m *model) addFields(d *definition, f *ast.Field) error {
	fieldType := f.Type
	if d.ModelsExternal {
		fieldType = rewriteLocalTypes(fieldType, d.ModelsPackageName).(ast.Expr)
	}
	typeString := sprintNode(fieldType)

	var (
		column                            string
		isPrimary, isExcluded, isReadOnly bool
	)
	if f.Tag != nil {
		// Parse the struct tag.
		// Example tag: `db:"column,primary,exclude,readonly"`.
		tag := reflect.StructTag(f.Tag.Value[1 : len(f.Tag.Value)-1]).Get("db")
		parts := strings.Split(tag, ",")
		column = parts[0]
		if column == "-" {
			return nil
		}
		for _, option := range parts[1:] {
			switch option {
			case "primary":
				isPrimary = true
			case "exclude":
				isExcluded = true
			case "readonly":
				isReadOnly = true
			}
		}
	}

	if f.Names == nil {
		if err := m.addEmbeddedStructFields(d, f); err == nil {
			return nil
		} else if err != errNoEmbeddedStruct {
			return err
		}
	}

	for _, name := range f.Names {
		typeName := name.Name
		if d.ModelsExternal && !ast.IsExported(typeName) {
			// Unexported type of external package, ignore because it isn't accessible.
			continue
		}

		spec := &field{
			Name:       typeName,
			Column:     column,
			Type:       typeString,
			PrimaryKey: isPrimary,
			Exclude:    isExcluded,
			ReadOnly:   isReadOnly,
		}
		if spec.Column == "" {
			spec.Column = spec.Name
		}

		if isPrimary {
			m.PrimaryKey = spec
		}

		m.fields = append(m.fields, spec)
	}

	return nil
}

var errNoEmbeddedStruct = errors.New("no embedded struct field")

// addEmbeddedStructFields adds any fields from an embedded struct field (flatten).
func (m *model) addEmbeddedStructFields(d *definition, f *ast.Field) error {
	ident, ok := f.Type.(*ast.Ident)
	if !ok || ident.Obj == nil {
		return errNoEmbeddedStruct
	}
	typeSpec, ok := ident.Obj.Decl.(*ast.TypeSpec)
	if !ok {
		return errNoEmbeddedStruct
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return errNoEmbeddedStruct
	}
	for _, field := range structType.Fields.List {
		if err := m.addFields(d, field); err != nil {
			return err
		}
	}
	return nil
}

// parseDocArgument finds a docstring parameter with name in doc, or returns
// an empty string.
//
// Example docstring parameter: `order by: id DESC`. The value can contain
// words, underscores and spaces. Values can also be quoted using a double
// quote. Quotes can be escaped using two double quotes.
func parseDocArgument(doc, name string) string {
	re := regexp.MustCompile(name + `:\s*([\w_ ]+|"(?:[^"]|"")*")`)
	if match := re.FindStringSubmatch(doc); len(match) == 2 {
		return strings.Trim(match[1], `"`)
	}
	return ""
}

// sprintNode returns the string (type) representation of the given node.
// Usaually an expression or type. For example, if n is a ast.TypeSpec
// of string, "string" is returned.
func sprintNode(n ast.Node) string {
	var buf strings.Builder
	if err := format.Node(&buf, token.NewFileSet(), n); err != nil {
		panic(err)
	}
	return buf.String()
}

// rewriteLocalTypes rewrites any (in the source) local exported types
// to imported types recursively.
func rewriteLocalTypes(n ast.Node, importName string) ast.Node {
	switch cur := n.(type) {
	case *ast.Ident:
		if cur.IsExported() {
			return &ast.SelectorExpr{
				X: &ast.Ident{
					Name: importName,
				},
				Sel: cur,
			}
		}

	case *ast.StarExpr:
		return &ast.StarExpr{
			X: rewriteLocalTypes(cur.X, importName).(ast.Expr),
		}

	case *ast.ArrayType:
		return &ast.ArrayType{
			Elt: rewriteLocalTypes(cur.Elt, importName).(ast.Expr),
			Len: cur.Len,
		}

	case *ast.MapType:
		return &ast.MapType{
			Key:   rewriteLocalTypes(cur.Key, importName).(ast.Expr),
			Value: rewriteLocalTypes(cur.Value, importName).(ast.Expr),
		}

	case *ast.InterfaceType:
		return &ast.InterfaceType{
			Methods:    rewriteLocalTypes(cur.Methods, importName).(*ast.FieldList),
			Incomplete: cur.Incomplete,
		}

	case *ast.StructType:
		return &ast.StructType{
			Fields:     rewriteLocalTypes(cur.Fields, importName).(*ast.FieldList),
			Incomplete: cur.Incomplete,
		}

	case *ast.FieldList:
		fieldList := &ast.FieldList{
			List: make([]*ast.Field, len(cur.List)),
		}
		for i, field := range cur.List {
			fieldList.List[i] = rewriteLocalTypes(field, importName).(*ast.Field)
		}
		return fieldList

	case *ast.Field:
		return &ast.Field{
			Doc:     cur.Doc,
			Names:   cur.Names,
			Type:    rewriteLocalTypes(cur.Type, importName).(ast.Expr),
			Tag:     cur.Tag,
			Comment: cur.Comment,
		}

	case *ast.FuncType:
		return &ast.FuncType{
			Params:  rewriteLocalTypes(cur.Params, importName).(*ast.FieldList),
			Results: rewriteLocalTypes(cur.Results, importName).(*ast.FieldList),
		}
	}

	return n
}
