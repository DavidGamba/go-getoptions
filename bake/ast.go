package main

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/DavidGamba/go-getoptions"
	"github.com/davecgh/go-spew/spew"
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

// The goal is to be able to find the getoptions.CommandFn calls.
// Also, we need to inspect the function and get the opt.<Type> calls to know what options are being used.
//
//	func Asciidoc(opt *getoptions.GetOpt) getoptions.CommandFn {
//		opt.String("lang", "en", opt.ValidValues("en", "es"))
//		opt.String("hello", "world")
//		opt.String("hola", "mundo")
//		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
func PrintAst(dir string) error {

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
			// Iterate through every node in the file
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				// Check function declarations for exported functions
				case *ast.FuncDecl:
					if x.Name.IsExported() {
						name := x.Name.Name
						description := x.Doc.Text()
						var buf bytes.Buffer
						printer.Fprint(&buf, fset, x.Type)
						Logger.Printf("file: %s, FuncDecl name: %s, desc: %s\n", file, name, strings.TrimSpace(description))
						Logger.Printf("file: %s, FuncDecl name: %s, type: %s\n", file, name, buf.String())

						// Check for Expressions of opt type
						ast.Inspect(n, func(n ast.Node) bool {
							switch x := n.(type) {
							case *ast.BlockStmt:
								for i, stmt := range x.List {
									Logger.Printf("i: %d\n", i)
									Logger.Printf("stmt: %T\n", stmt)
									// We are expecting the expression before the return function
									_, ok := stmt.(*ast.ReturnStmt)
									if ok {
										return false
									}
									Logger.Printf("inspect stmt: %T\n", stmt)
									exprStmt, ok := stmt.(*ast.ExprStmt)
									if !ok {
										continue
									}
									// spew.Dump(exprStmt)

									// Check for CallExpr
									ast.Inspect(exprStmt, func(n ast.Node) bool {
										switch x := n.(type) {
										case *ast.CallExpr:
											fun, ok := x.Fun.(*ast.SelectorExpr)
											if !ok {
												return false
											}
											xIdent, ok := fun.X.(*ast.Ident)
											if !ok {
												return false
											}
											xSel := fun.Sel.Name
											Logger.Printf("X: %s, Selector: %s\n", xIdent.Name, xSel)

											// Check for args
											for _, arg := range x.Args {
												Logger.Printf("arg: %T\n", arg)
												spew.Dump(arg)
											}
											return false
										}
										return true
									})
								}
							}
							return true
						})
					}
				}
				return true
			})
		}
	}
	return nil
}

// The goal is to be able to find the getoptions.CommandFn calls.
// Also, we need to inspect the function and get the opt.<Type> calls to know what options are being used.
//
//	func Asciidoc(opt *getoptions.GetOpt) getoptions.CommandFn {
//		opt.String("lang", "en", opt.ValidValues("en", "es"))
//		opt.String("hello", "world")
//		opt.String("hola", "mundo")
//		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
func LoadAst(ctx context.Context, opt *getoptions.GetOpt, dir string) error {
	// Regex for description: fn-name - description
	re := regexp.MustCompile(`^\w\S+ -`)
	ot := NewOptTree(opt)

	for p, err := range parsedFiles(dir) {
		if err != nil {
			return err
		}

		// Iterate through every node in the file
		ast.Inspect(p.f, func(n ast.Node) bool {
			switch x := n.(type) {
			// Check function declarations for exported functions
			case *ast.FuncDecl:
				if x.Name.IsExported() {
					name := x.Name.Name
					description := strings.TrimSpace(x.Doc.Text())
					// var buf bytes.Buffer
					// printer.Fprint(&buf, p.fset, x.Type)
					// Logger.Printf("file: %s\n", p.file)
					// Logger.Printf("type: %s, name: %s, desc: %s\n", buf.String(), name, description)

					// Expect function of type:
					// func Name(opt *getoptions.GetOpt) getoptions.CommandFn

					// Check Params
					// Expect opt *getoptions.GetOpt
					if len(x.Type.Params.List) != 1 {
						return false
					}
					var optFieldName string
					for _, param := range x.Type.Params.List {
						name := param.Names[0].Name
						var buf bytes.Buffer
						printer.Fprint(&buf, p.fset, param.Type)
						// Logger.Printf("name: %s, %s\n", name, buf.String())
						if buf.String() != "*getoptions.GetOpt" {
							return false
						}
						optFieldName = name
					}

					// Check Results
					// Expect getoptions.CommandFn
					if len(x.Type.Results.List) != 1 {
						return false
					}
					for _, result := range x.Type.Results.List {
						var buf bytes.Buffer
						printer.Fprint(&buf, p.fset, result.Type)
						// Logger.Printf("result: %s\n", buf.String())
						if buf.String() != "getoptions.CommandFn" {
							return false
						}
					}

					// TODO: The yield probably goes here
					// Add function to OptTree
					if description != "" {
						// Logger.Printf("description '%s'\n", description)
						if re.MatchString(description) {
							// Get first word from string
							name = strings.Split(description, " ")[0]
							description = strings.TrimPrefix(description, name+" -")
							description = strings.TrimSpace(description)
						}
					} else {
						name = camelToKebab(name)
					}
					cmd := ot.AddCommand(name, description)

					// Check for Expressions of opt type
					ast.Inspect(n, func(n ast.Node) bool {
						switch x := n.(type) {
						case *ast.BlockStmt:
							for _, stmt := range x.List {
								var buf bytes.Buffer
								printer.Fprint(&buf, p.fset, stmt)
								// We are expecting the expression before the return function
								_, ok := stmt.(*ast.ReturnStmt)
								if ok {
									return false
								}
								// Logger.Printf("stmt: %s\n", buf.String())
								exprStmt, ok := stmt.(*ast.ExprStmt)
								if !ok {
									continue
								}
								// spew.Dump(exprStmt)

								// Check for CallExpr
								ast.Inspect(exprStmt, func(n ast.Node) bool {
									switch x := n.(type) {
									case *ast.CallExpr:
										fun, ok := x.Fun.(*ast.SelectorExpr)
										if !ok {
											return false
										}
										xIdent, ok := fun.X.(*ast.Ident)
										if !ok {
											return false
										}
										if xIdent.Name != optFieldName {
											return false
										}
										// Logger.Printf("handling %s.%s\n", xIdent.Name, fun.Sel.Name)

										switch fun.Sel.Name {
										case "String":
											handleString(cmd, optFieldName, n)
										}

										return false
									}
									return true
								})
							}
						}
						return true
					})
				}
			}
			return true
		})
	}
	return nil
}

func handleString(cmd *getoptions.GetOpt, optFieldName string, n ast.Node) error {
	x := n.(*ast.CallExpr)
	name := ""
	defaultValue := ""
	mfns := []getoptions.ModifyFn{}
	// Check for args
	for i, arg := range x.Args {
		// Logger.Printf("i: %d, arg: %T\n", i, arg)
		if i == 0 {
			// First argument is the Name
			// Logger.Printf("Name: '%s'\n", arg.(*ast.BasicLit).Value)
			var err error
			name, err = strconv.Unquote(arg.(*ast.BasicLit).Value)
			if err != nil {
				name = arg.(*ast.BasicLit).Value
			}
		} else if i == 1 {
			// Second argument is the Default
			// Logger.Printf("Default: '%s'\n", arg.(*ast.BasicLit).Value)
			var err error
			defaultValue, err = strconv.Unquote(arg.(*ast.BasicLit).Value)
			if err != nil {
				defaultValue = arg.(*ast.BasicLit).Value
			}
		} else {
			// Remaining arguments are option modifiers

			callE, ok := arg.(*ast.CallExpr)
			if !ok {
				continue
			}
			fun, ok := callE.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			xIdent, ok := fun.X.(*ast.Ident)
			if !ok {
				continue
			}
			if xIdent.Name != optFieldName {
				continue
			}
			// Logger.Printf("\t%s.%s\n", xIdent.Name, fun.Sel.Name)
			if fun.Sel.Name == "SetCalled" {
				// TODO: SetCalled function receives a bool
				continue
			}
			values := []string{}
			for _, arg := range callE.Args {
				// Logger.Printf("Value: %s\n", arg.(*ast.BasicLit).Value)
				value, err := strconv.Unquote(arg.(*ast.BasicLit).Value)
				if err != nil {
					value = arg.(*ast.BasicLit).Value
				}
				values = append(values, value)
			}
			switch fun.Sel.Name {
			case "Alias":
				mfns = append(mfns, cmd.Alias(values...))
			case "ArgName":
				if len(values) > 0 {
					mfns = append(mfns, cmd.ArgName(values[0]))
				}
			case "Description":
				if len(values) > 0 {
					mfns = append(mfns, cmd.Description(values[0]))
				}
			case "GetEnv":
				if len(values) > 0 {
					mfns = append(mfns, cmd.GetEnv(values[0]))
				}
			case "Required":
				mfns = append(mfns, cmd.Required(values...))
			case "SuggestedValues":
				mfns = append(mfns, cmd.SuggestedValues(values...))
			case "ValidValues":
				mfns = append(mfns, cmd.ValidValues(values...))
			}
		}
	}
	cmd.String(name, defaultValue, mfns...)
	return nil
}
