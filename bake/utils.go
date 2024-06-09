package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"iter"

	"golang.org/x/tools/go/packages"
)

type parsedFile struct {
	file string
	fset *token.FileSet
	f    *ast.File
}

// Requires GOEXPERIMENT=rangefunc
func parsedFiles(dir string) iter.Seq2[parsedFile, error] {
	return func(yield func(parsedFile, error) bool) {
		cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax, Dir: dir}
		pkgs, err := packages.Load(cfg, ".")
		if err != nil {
			yield(parsedFile{}, fmt.Errorf("failed to load packages: %w", err))
			return
		}
		for _, pkg := range pkgs {
			// Logger.Println(pkg.ID, pkg.GoFiles)
			for _, file := range pkg.GoFiles {
				p := parsedFile{}
				// Logger.Printf("file: %s\n", file)
				// parse file
				fset := token.NewFileSet()
				fset.AddFile(file, fset.Base(), len(file))
				p.file = file
				p.fset = fset
				f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
				if err != nil {
					yield(p, fmt.Errorf("failed to parse file: %w", err))
				}
				p.f = f
				yield(p, nil)
			}
		}
	}
}
