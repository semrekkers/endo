package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"go/format"
	"os"
	"text/template"

	"gopkg.in/yaml.v2"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	var (
		argSchema = flag.String("schema", "stdin", "Schema `file` to use")
		argModels = flag.Bool("models", false, "Generate models instead of repository")
		argOutput = flag.String("output", "stdout", "Output `file`")
	)
	flag.Parse()

	schemaFile, schemaClose, err := openFile(*argSchema, false)
	exitOnErr(err)
	defer schemaClose()

	var schema Schema
	dec := yaml.NewDecoder(schemaFile)
	dec.SetStrict(true)
	err = dec.Decode(&schema)
	exitOnErr(err)
	exitOnErr(setDefaults(&schema))

	templates := template.New("endogen")
	templates.Funcs(template.FuncMap{
		"joinStrings": joinStrings,
		"mapToParams": mapToParams,
	})
	_, err = templates.ParseFS(templateFS, "templates/*")
	exitOnErr(err)

	var (
		buf    bytes.Buffer
		output []byte
	)
	templateName := "repository.go.tmpl"
	if *argModels {
		templateName = "models.go.tmpl"
	}
	exitOnErr(templates.ExecuteTemplate(&buf, templateName, &schema))
	output, err = format.Source(buf.Bytes())
	exitOnErr(err)

	outputFile, outputClose, err := openFile(*argOutput, true)
	exitOnErr(err)
	defer outputClose()
	_, err = outputFile.Write(output)
	exitOnErr(err)
}

func openFile(name string, create bool) (file *os.File, close func() error, err error) {
	close = nilClose
	switch name {
	case "stdin", "-":
		file = os.Stdin
	case "stdout":
		file = os.Stdout
	case "stderr":
		file = os.Stderr
	default:
		if create {
			file, err = os.Create(name)
		} else {
			file, err = os.Open(name)
		}
		if err != nil {
			return
		}
		close = file.Close
	}
	return
}

func nilClose() error {
	return nil
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
