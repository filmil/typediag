// Package main is the entrypoint for the program "typediag", which prints a
// dot digraph of a go package.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

var (
	// path is the go package path to analyze.
	path string

	exportedOnly bool
)

type diag struct {
	exported bool
	err      error
	fset     token.FileSet
}

// ParseDir parses the go package in the given path.
func (d *diag) ParseDir(path string) map[string]*ast.Package {
	if d.err != nil {
		return nil
	}
	var p map[string]*ast.Package
	p, d.err = parser.ParseDir(&d.fset, path, nil /*FilterFunc*/, parser.AllErrors)
	return p
}

func (d *diag) Sprint(n ast.Node) string {
	if d.err != nil {
		return ""
	}
	var b strings.Builder
	d.err = printer.Fprint(&b, &d.fset, n)
	return strings.TrimLeft(b.String(), "*")
}

type Function struct {
	Package                 string
	File                    string
	Name                    string
	Recv                    string
	ParamTypes, ReturnTypes map[string]bool
}

func NewFunction(pkg, file, name, recv string) *Function {
	return &Function{pkg, file, name, recv, map[string]bool{}, map[string]bool{}}
}

func (f Function) String() string {
	if f.Recv != "" {
		return fmt.Sprintf("func (%v) '%v'.%v", f.Recv, f.Package, f.Name)
	}
	return fmt.Sprintf("func '%v'.%v", f.Package, f.Name)
}

type Info struct {
	Functions []Function
	Types     map[string]bool
}

var (
	dotTpl = template.Must(template.New("digraph").Parse(`
digraph G {
  /* Types */
  {{range $key, $val := .Types }}"{{$key}}";
  {{end}}

  /* Functions and type connectivity */
  {{range $i, $fun := .Functions}} f{{$i}} [shape=box,label="{{$fun}}"];
    {{range $pt, $_ := .ParamTypes }}
	  "{{$pt}}"->f{{$i}};
	{{end}}
	{{if ne .Recv ""}}
	  "{{.Recv}}"->f{{$i}}; 
	{{end}}
    {{range $rt, $_ := .ReturnTypes }}
	  f{{$i}}->"{{$rt}}";
	{{end}}
  {{end}}
}
`))
)

func (i Info) Render(w io.Writer) error {
	return dotTpl.Execute(w, i)
}

func NewInfo() *Info {
	return &Info{Types: map[string]bool{}}
}

func (d *diag) RecvType(r *ast.FieldList) string {
	if d.err != nil {
		return ""
	}
	if r == nil {
		return ""
	}
	recvType := r.List[0].Type
	return d.Sprint(recvType)
}

func (d *diag) FuncDecls(packages map[string]*ast.Package) *Info {
	if d.err != nil {
		return nil
	}
	i := NewInfo()
	for pname, pkg := range packages {
		for fname, file := range pkg.Files {
			scope := file.Scope
			if scope != nil {
				for objname, obj := range scope.Objects {
					if obj.Kind == ast.Typ {
						if d.exported && !ast.IsExported(obj.Name) {
							continue
						}
						i.Types[objname] = true
					}
				}
			}
			for _, decl := range file.Decls {
				switch decl.(type) {
				default:
					continue
				case *ast.FuncDecl:
					fd := decl.(*ast.FuncDecl)
					fun := NewFunction(pname, fname, fd.Name.Name, d.RecvType(fd.Recv))
					ft := fd.Type
					if ft.Params != nil {
						for _, field := range ft.Params.List {
							typeStr := d.Sprint(field.Type)
							if d.exported && !ast.IsExported(typeStr) {
								continue
							}
							// Add parameter type to list of all types.
							i.Types[typeStr] = true
							// Add parameter type to list of parameter types.
							fun.ParamTypes[typeStr] = true
						}
					}
					if ft.Results != nil {
						for _, field := range ft.Results.List {
							typeStr := d.Sprint(field.Type)
							if d.exported && !ast.IsExported(typeStr) {
								continue
							}
							i.Types[typeStr] = true
							fun.ReturnTypes[typeStr] = true
						}
					}
					i.Functions = append(i.Functions, *fun)
				}
			}

		}
	}
	return i
}

func (d *diag) RenderDot(decls *Info, w io.Writer) {
	if d.err != nil {
		return
	}
	d.err = decls.Render(w)
}

func main() {
	var d diag
	flag.StringVar(&path, "path", "", "Package path to analyze")
	flag.BoolVar(&d.exported, "exported-only", true, "If set, only exported types are analyzed")
	flag.Parse()

	if path == "" {
		log.Fatalf("flag --path=... is required")
	}

	p := d.ParseDir(path)
	decls := d.FuncDecls(p)
	d.RenderDot(decls, os.Stdout)
	if d.err != nil {
		log.Fatalf("unexpected error: %v", d.err)
	}
}
