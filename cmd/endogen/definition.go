package main

import (
	"errors"
	"fmt"
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
	Imports             []*importInfo
	PatchTypeMode       string
	ModelsExternal      bool
	ModelsPackageName   string
	ModelsPackagePrefix string
	Store               string
	GenerateStore       bool
	ReadOnly            bool
	Models              []*model
}

type importInfo struct {
	Name, Path string
}

func (inf *importInfo) Spec() string {
	return fmt.Sprintf("%s %q", inf.Name, inf.Path)
}

// model repressets a single model (table or view) to generate code for.
type model struct {
	Name          string // model name in source code
	PackagePrefix string // package import name prefix of model, or empty when local
	Type          string // model type in source code
	Patches       string // if not empty: this is an patch type model for <Patches>
	Generate      bool   // whether this model must be generated
	ReadOnly      bool   // model is read-only
	Immutable     bool   // model is immutable (only create, no updates)
	Plural        string // plural of name
	Table         string // table name in database
	Sort          string // sort order to use for result set, if any

	Patch *model // patch type of this model

	fields []*field
}

type field struct {
	Name     string // field name in source code
	Column   string // column name in model
	Type     string // field type in source code
	ReadOnly bool   // whether this field is read-only
}

// Fields returns the fields of the model. If forWrite is true, only
// writable fields are returned.
func (m *model) Fields(forWrite bool) []*field {
	fields, n := make([]*field, len(m.fields)), 0
	for _, field := range m.fields {
		if forWrite && field.ReadOnly {
			continue
		}
		fields[n] = field
		n++
	}
	return fields[:n]
}

// Updatable returns whether m is updatable by patch or replacement.
func (m *model) Updatable() bool {
	return !(m.ReadOnly || m.Immutable || m.Patches != "")
}

func (d *definition) addImport(name, path string) {
	for _, imported := range d.Imports {
		if path == imported.Path {
			return
		}
	}
	d.Imports = append(d.Imports, &importInfo{
		Name: name,
		Path: path,
	})
}

func (d *definition) addImports(a []*ast.ImportSpec) {
	for _, v := range a {
		var (
			name    string
			path, _ = strconv.Unquote(v.Path.Value)
		)
		if v.Name != nil {
			name = v.Name.Name
		}
		d.addImport(name, path)
	}
}

// addFile parses the given file and adds the models it defines to
// the definition.
func (d *definition) addFile(f *ast.File) error {
	d.addImports(f.Imports)
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

			if err := d.addModel(typeName, genDecl.Doc.Text(), structType); err != nil {
				return err
			}
		}
	}

	return nil
}

// addModel parses the given StructType as model.
func (d *definition) addModel(name, comment string, s *ast.StructType) error {
	if strings.Contains(comment, "endo-ignore") {
		return nil
	}
	args := parseCommentArguments(comment)
	m := model{
		Name:          name,
		PackagePrefix: d.ModelsPackagePrefix,
		Type:          name,
		Patches:       args["patches"],
		ReadOnly:      d.ReadOnly,
		Plural:        args["plural"],
		Table:         args["table"],
		Sort:          args["sort"],
	}

	if arg := args["read-only"]; arg != "" {
		m.ReadOnly, _ = strconv.ParseBool(arg)
	}
	if arg := args["immutable"]; arg != "" {
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

	for _, field := range s.Fields.List {
		c := field
		m.addFields(d, c)
	}

	d.Models = append(d.Models, &m)
	return nil
}

func (d *definition) resolveModelDependencies(createMissing bool) error {
	var (
		baseTypes  []*model
		patchTypes []*model
	)
	// Split the models into base and patch types, we want to keep the order.
	for _, m := range d.Models {
		if m.Patches != "" {
			patchTypes = append(patchTypes, m)
		} else {
			baseTypes = append(baseTypes, m)
		}
	}
	// Assign each patch type to it's base.
	for _, patchType := range patchTypes {
		for _, m := range baseTypes {
			if patchType.Patches == m.Type {
				if !m.Updatable() {
					return fmt.Errorf("%s: cannot patch a non-updatable type (%s)", patchType.Type, m.Type)
				}
				if m.Patch != nil {
					return fmt.Errorf("%s: type (%s) already has a patch type (%s), only one can be assigned", patchType.Type, m.Type, m.Patch.Type)
				}
				m.Patch = patchType
			}
		}
	}
	// Check missing patch types.
	for _, m := range baseTypes {
		if !m.Updatable() || m.Patch != nil {
			continue
		}
		if !createMissing {
			return fmt.Errorf("type (%s) has no patch type but requires one", m.Type)
		}
		m.Patch = d.newPatchTypeOf(m)
	}

	d.Models = baseTypes
	return nil
}

func (d *definition) newPatchTypeOf(b *model) *model {
	name := b.Type + "Patch"
	m := &model{
		Generate: true,
		Name:     name,
		Type:     name,
		Patches:  b.Type,
	}
	for _, bField := range b.fields {
		if bField.ReadOnly {
			continue
		}
		m.fields = append(m.fields, &field{
			Name:   bField.Name,
			Column: bField.Column,
			Type:   "*" + bField.Type, // pointer type
		})
	}
	return m
}

func (m *model) addFields(d *definition, f *ast.Field) error {
	fieldType := f.Type
	if d.ModelsExternal {
		fieldType = rewriteLocalTypes(fieldType, d.ModelsPackageName).(ast.Expr)
	}
	typeString := sprintNode(fieldType)

	var (
		column         string
		readOnly, sort bool
	)
	if f.Tag != nil {
		// Parse the struct tag.
		// Example tag: `db:"column,readonly,sort"`.
		tag := reflect.StructTag(f.Tag.Value[1 : len(f.Tag.Value)-1]).Get("db")
		parts := strings.Split(tag, ",")
		column = parts[0]
		if column == "-" {
			return nil
		}
		for _, option := range parts[1:] {
			switch option {
			case "readonly":
				readOnly = true
			case "sort":
				sort = true
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
			Name:     typeName,
			Column:   column,
			Type:     typeString,
			ReadOnly: readOnly,
		}
		if spec.Column == "" {
			spec.Column = spec.Name
		}

		if sort && m.Sort == "" {
			m.Sort = spec.Column
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

var commentArgumentRegex = regexp.MustCompile(`(\w[\w- ]+):\s*(\w[\w_ ]*|"(?:[^"]|"")*")`)

// parseCommentArguments finds defined parameters in comment.
//
// Example parameter: `order by: id DESC`. The value can contain
// words, underscores and spaces. Values can also be quoted using a double
// quote. Quotes can be escaped using two double quotes.
func parseCommentArguments(comment string) (res map[string]string) {
	res = make(map[string]string)
	for _, match := range commentArgumentRegex.FindAllStringSubmatch(comment, -1) {
		v := strings.Trim(match[2], `"`)
		res[match[1]] = strings.ReplaceAll(v, `""`, `"`)
	}
	return res
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
