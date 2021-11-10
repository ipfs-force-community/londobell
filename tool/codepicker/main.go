package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"text/template"
)

type pickTarget struct {
	pkg   string
	names []string
	t     string
	dest  string
}

func (pt *pickTarget) gen(buf *bytes.Buffer) error {
	namesCount := len(pt.names)
	if namesCount == 0 {
		return fmt.Errorf("no target names specified")
	}

	buf.Reset()

	pkg, err := build.Import(pt.pkg, ".", build.FindOnly)
	if err != nil {
		return fmt.Errorf("find package: %w", err)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkg.Dir, nil, 0)
	if err != nil {
		return fmt.Errorf("parse package: %w", err)
	}

	var found ast.Node

	for name, p := range pkgs {
		log.Println("inspect", name)
		ast.Inspect(p, func(node ast.Node) bool {
		FIND:
			switch inner := node.(type) {
			case *ast.FuncDecl:
				if inner.Name != nil && namesCount == 1 && pt.names[0] == inner.Name.Name {
					found = node
					return false
				}

			case *ast.GenDecl:
				if inner.Tok == token.CONST || inner.Tok == token.VAR {
					matched := 0
					for _, spec := range inner.Specs {
						v, ok := spec.(*ast.ValueSpec)
						if !ok {
							break
						}

						for _, name := range v.Names {
							// we have too many spec names to compare
							if matched >= namesCount {
								break FIND
							}
							if name.Name != pt.names[matched] {
								break FIND
							}

							matched++
						}
					}

					if matched == namesCount {
						found = node
						return false
					}

					return true
				}
			}

			return true
		})
	}

	if found == nil {
		return fmt.Errorf("func not found")
	}

	err = format.Node(buf, fset, found)
	if err != nil {
		return fmt.Errorf("format func: %w", err)
	}

	t, err := template.ParseFiles(pt.t)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	content := buf.String()
	buf.Reset()

	buf.Write([]byte("// GENERATED, DO NOT EDIT\n"))

	err = t.Execute(buf, map[string]interface{}{
		"Code":  content,
		"Names": pt.names,
	})
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}

	err = os.WriteFile(pt.dest, formatted, 0644)
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func main() {
	picks := []pickTarget{
		{
			pkg:   "github.com/filecoin-project/lotus/node",
			names: []string{"handleFractionOpt"},
			t:     "racailum/debug/set_handler.go.template",
			dest:  "racailum/debug/set_handler.go",
		},
		{
			pkg: "github.com/filecoin-project/lotus/lib/tracing",
			names: []string{
				"envCollectorEndpoint",
				"envAgentEndpoint",
				"envAgentHost",
				"envAgentPort",
				"envJaegerUser",
				"envJaegerCred",
			},
			t:    "racailum/tracing/options.go.template",
			dest: "racailum/tracing/options.go",
		},
	}

	var buf bytes.Buffer
	for _, pick := range picks {
		desc := fmt.Sprintf("pick code for %v from %s to file %s", pick.names, pick.pkg, pick.dest)
		err := pick.gen(&buf)
		if err != nil {
			log.Fatalf("%s: %s", desc, err)
		}

		log.Println(desc, "done")
	}
}
