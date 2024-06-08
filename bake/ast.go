package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"golang.org/x/tools/go/packages"
)

type FuncDecl struct {
	Description string
	Type        string
}

// m: map of function name to function information
func GetFuncDeclForPackage(dir string, m *map[string]FuncDecl) error {
	if m == nil {
		return fmt.Errorf("map is nil")
	}
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax, Dir: dir}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}
	for _, pkg := range pkgs {
		// Logger.Println(pkg.ID, pkg.GoFiles)
		for _, file := range pkg.GoFiles {
			// Logger.Printf("file: %s\n", file)
			// parse file
			fset := token.NewFileSet()
			fset.AddFile(file, fset.Base(), len(file))
			f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("failed to parse file: %w", err)
			}
			// inspect file
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					if x.Name.IsExported() {
						name := x.Name.Name
						description := x.Doc.Text()
						var buf bytes.Buffer
						printer.Fprint(&buf, fset, x.Type)
						Logger.Printf("file: %s, name: %s, type: %s\n", file, name, buf.String())
						(*m)[name] = FuncDecl{
							Description: description,
							Type:        buf.String(),
						}
					}
				}
				return true
			})
		}
	}
	return nil
}
