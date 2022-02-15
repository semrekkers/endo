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

const (
	patchTypeModeInclude = "include"
	patchTypeModeOnly    = "only"
	patchTypeModeImport  = "import"
)

func main() {
	var (
		argImportPath    = flag.String("import", ".", "Import `path` of the input")
		argViews         = flag.Bool("views", false, "Treat model structs as read-only views")
		argPatchTypeMode = flag.String("patch", "include", "Patch type generation `mode` [include, only, import]")
		argStoreType     = flag.String("store-type", "Store", "The `type name` to use for the store")
		argGenStore      = flag.Bool("gen-store", true, "Also generate the store type and constructor")
		argPkgName       = flag.String("pkg", "", "Package `name` to use in the output (default use package name from input)")
		argImportAlias   = flag.String("import-alias", "", "Alias `name` to use for the imported external package of input")
		argOutput        = flag.String("out", "", "Output `file` to write the result to (default writes to stdout)")
	)
	flag.Parse()
	switch *argPatchTypeMode {
	case patchTypeModeInclude, patchTypeModeOnly, patchTypeModeImport:
		break

	default:
		exitOnErr(fmt.Errorf("patch flag value can only be one of: include, only or import, not %s", *argPatchTypeMode))
	}

	var (
		fset          = token.NewFileSet()
		inputFileName = os.ExpandEnv("$GOFILE")
		source        *ast.File
		importDir     = *argImportPath
		err           error
	)
	if firstArg := flag.Arg(0); firstArg != "" {
		inputFileName = firstArg
	}
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
		Imports:       baseImports,
		PatchTypeMode: *argPatchTypeMode,
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
		d.addImport(*argImportAlias, pkgs[0].PkgPath)
		d.ModelsExternal = true
		d.ModelsPackageName = pkgName
		d.ModelsPackagePrefix = pkgName + "." // so that it corresponds to modelPackage.ModelType
	}

	exitOnErr(d.addFile(source))
	if inputFileArgs := flag.Args(); 1 < len(inputFileArgs) {
		// Add subsequent source files
		for _, inputFileName := range inputFileArgs[1:] {
			source, err = parser.ParseFile(fset, inputFileName, nil, parser.ParseComments)
			exitOnErr(err)
			exitOnErr(d.addFile(source))
		}
	}
	exitOnErr(d.resolveModelDependencies())

	var (
		templates   = getTemplates()
		runTemplate = "store.go.tmpl"
		buf         bytes.Buffer
	)
	if *argPatchTypeMode == patchTypeModeOnly {
		runTemplate = "patchtype.go.tmpl"
	}
	exitOnErr(templates.ExecuteTemplate(&buf, runTemplate, &d))
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

var baseImports = []*importInfo{
	{Path: "context"},
	{Path: "database/sql"},
	{Path: "github.com/lib/pq"},
	{Path: "github.com/semrekkers/endo/pkg/endo"},
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

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
