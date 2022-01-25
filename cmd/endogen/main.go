package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

var (
	fset      = token.NewFileSet()
	templates = template.New("endogen")
)

func init() {
	templates.Funcs(template.FuncMap{
		"filterPrimary":  filterPrimary,
		"toColumns":      toColumns,
		"joinStrings":    joinStrings,
		"mapToParams":    mapToParams,
		"lastArg":        lastArg,
		"toFieldUpdates": toFieldUpdates,
	})
	_, err := templates.ParseFS(templateFS, "templates/*")
	if err != nil {
		panic(err)
	}
}

func main() {
	var (
		argInput     = flag.String("in", "$GOFILE", "Input Go `file` containing the model structs ('stdin' or empty reads from stdin)")
		argViews     = flag.Bool("views", false, "Treat model structs as read-only views")
		argStoreType = flag.String("store-type", "Store", "The `type name` to use for the store")
		argGenStore  = flag.Bool("gen-store", true, "Generate the Store type and constructor")
		argOutput    = flag.String("out", "stdout", "Output `file` to write the result to ('stdout' writes to stdout)")
	)
	flag.Parse()

	var (
		source *ast.File
		err    error
	)
	inputFileName := os.ExpandEnv(*argInput)
	if inputFileName == "" || inputFileName == "stdin" {
		source, err = parser.ParseFile(fset, "", os.Stdin, parser.ParseComments)
	} else {
		source, err = parser.ParseFile(fset, inputFileName, nil, parser.ParseComments)
	}
	exitOnErr(err)

	d := definition{
		Package:       source.Name.String(),
		ExtraImports:  getExtraImportSpecs(source.Imports),
		Store:         *argStoreType,
		GenerateStore: *argGenStore,
		ReadOnly:      *argViews,
	}
	exitOnErr(d.addFile(source))

	var buf bytes.Buffer
	exitOnErr(templates.ExecuteTemplate(&buf, "store.go.tmpl", &d))
	result, err := format.Source(buf.Bytes())
	exitOnErr(err)

	output := os.Stdout
	if *argOutput != "stdout" {
		output, err = os.Create(*argOutput)
		exitOnErr(err)
		defer output.Close()
	}
	_, err = output.Write(result)
	exitOnErr(err)
}

// ignoreImportPaths is a list of import paths that should be ignored because
// they are already imported in the generated code.
var ignoreImportPaths = []string{
	`"context"`,
	`"database/sql"`,
	`"github.com/lib/pq"`,
	`"github.com/semrekkers/endo/pkg/endo"`,
}

func getExtraImportSpecs(imports []*ast.ImportSpec) []string {
	var result []string

outer:
	for _, extraImport := range imports {
		var path, name string
		if extraImport.Path == nil {
			continue
		}
		path = extraImport.Path.Value
		for _, ignorePath := range ignoreImportPaths {
			if path == ignorePath {
				continue outer
			}
		}
		if extraImport.Name != nil {
			name = extraImport.Name.Name
		}
		result = append(result, name+" "+path)
	}

	return result
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
