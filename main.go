package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ardanlabs/ffi-converter/generator"
	"github.com/ardanlabs/ffi-converter/parser"
)

func main() {
	headerPath := flag.String("header", "", "Path to C header file")
	outputDir := flag.String("output", ".", "Output directory for generated Go files")
	packageName := flag.String("package", "bindings", "Go package name")
	libName := flag.String("lib", "", "Library name (e.g., 'mylib' for libmylib.so)")
	flag.Parse()

	if *headerPath == "" {
		fmt.Fprintln(os.Stderr, "error: -header flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *libName == "" {
		base := filepath.Base(*headerPath)
		ext := filepath.Ext(base)
		*libName = base[:len(base)-len(ext)]
	}

	headerData, err := os.ReadFile(*headerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading header: %v\n", err)
		os.Exit(1)
	}

	header, err := parser.Parse(string(headerData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing header: %v\n", err)
		os.Exit(1)
	}

	gen := generator.New(*packageName, *libName, header)

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	files, err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(1)
	}

	for filename, content := range files {
		path := filepath.Join(*outputDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", path)
	}
}
