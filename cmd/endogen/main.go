package main

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"text/template"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	var (
		argInput       = flag.String("in", "$GOFILE", "Input Go `file` containing the model structs ('stdin' or empty reads from stdin)")
		argImportPath  = flag.String("import", ".", "Import `path` of the input")
		argViews       = flag.Bool("views", false, "Treat model structs as read-only views")
		argStoreType   = flag.String("store-type", "Store", "The `type name` to use for the store")
		argGenStore    = flag.Bool("gen-store", true, "Also generate the store type and constructor")
		argPkgName     = flag.String("pkg", "", "Package `name` to use in the output (default use package name from input)")
		argImportAlias = flag.String("import-alias", "", "Alias `name` to use for the imported external package of input")
		argOutput      = flag.String("out", "", "Output `file` to write the result to (default writes to stdout)")
	)
	flag.Parse()

	var (
		fset      = token.NewFileSet()
		source    *ast.File
		importDir = *argImportPath
		err       error
	)
	inputFileName := os.ExpandEnv(*argInput)
	if inputFileName == "" || inputFileName == "stdin" {
		source, err = parser.ParseFile(fset, "", os.Stdin, parser.ParseComments)
	} else {
		source, err = parser.ParseFile(fset, inputFileName, nil, parser.ParseComments)
		if importDir == "." {
			importDir = filepath.FromSlash("./") + filepath.Dir(inputFileName)
		}
	}
	exitOnErr(err)

	sourcePackageName := source.Name.String()
	d := definition{
		Package:       sourcePackageName,
		ExtraImports:  getExtraImportSpecs(source.Imports),
		Store:         *argStoreType,
		GenerateStore: *argGenStore,
		ReadOnly:      *argViews,
	}
	if *argPkgName != "" {
		d.Package = *argPkgName
	}
	if d.Package != sourcePackageName {
		if importDir == "." {
			exitOnErr(errors.New("import path cannot be the same when the input and output package names differ"))
		}

		// Output is an external package, get the import path and name of the source package.
		pkgs, err := packages.Load(nil, importDir)
		exitOnErr(err)

		pkgName := pkgs[0].Name
		if *argImportAlias != "" {
			pkgName = *argImportAlias
		}
		d.ModelsExternal = true
		d.ModelsImportPath = pkgs[0].PkgPath
		d.ModelsImportAlias = *argImportAlias
		d.ModelsPackageName = pkgName
		d.ModelsPackagePrefix = pkgName + "." // so that it corresponds to modelPackage.ModelType
	}

	exitOnErr(d.addFile(source))

	var (
		templates = getTemplates()
		buf       bytes.Buffer
	)
	exitOnErr(templates.ExecuteTemplate(&buf, "store.go.tmpl", &d))
	result, err := imports.Process(*argOutput, buf.Bytes(), nil)
	exitOnErr(err)

	output := os.Stdout
	if *argOutput != "" {
		output, err = os.Create(*argOutput)
		exitOnErr(err)
		defer output.Close()
	}
	_, err = output.Write(result)
	exitOnErr(err)
}

func getTemplates() *template.Template {
	v := template.New("endogen")
	v.Funcs(template.FuncMap{
		"toColumns":      toColumns,
		"joinStrings":    joinStrings,
		"mapToParams":    mapToParams,
		"lastArg":        lastArg,
		"toFieldUpdates": toFieldUpdates,
	})
	_, err := v.ParseFS(templateFS, "templates/*")
	if err != nil {
		panic(err)
	}
	return v
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
