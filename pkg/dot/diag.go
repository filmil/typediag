// Package dot has code that generates dot graphs.
package dot

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"strings"
	"text/template"
)

// Diagram is a dot diagram genrerator based on passed in golang package path.
type Diagram struct {
	path     string
	exported bool
	err      error
	fset     token.FileSet
}

func NewDiagram(exported bool, path string) *Diagram {
	return &Diagram{path: path, exported: exported}
}

// ParseDir parses the go package in the given path.
func (d *Diagram) ParseDir(path string) map[string]*ast.Package {
	if d.err != nil {
		return nil
	}
	var p map[string]*ast.Package
	p, d.err = parser.ParseDir(&d.fset, path, nil /*FilterFunc*/, parser.AllErrors)
	return p
}

func (d *Diagram) Sprint(n ast.Node) string {
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
		return fmt.Sprintf("(%v)'%v'.%v", f.Recv, f.Package, f.Name)
	}
	return fmt.Sprintf("'%v'.%v", f.Package, f.Name)
}

type Info struct {
	Functions []Function
	Types     map[string]bool
}

var (
	dotTpl = template.Must(template.New("digraph").
		Funcs(map[string]interface{}{
			"constr":   constr,
			"exported": ast.IsExported,
		}).
		Parse(`
digraph G {
  /* Types */
  {{range $key, $val := .Types -}}
  "{{$key -}}";
  {{end}}

  /* Functions and type connectivity */
  {{range $i, $fun := .Functions}} f{{$i}} [shape=box,label="{{$fun}}"];
    {{range $pt, $_ := .ParamTypes -}}
	  "{{$pt}}"->f{{$i}};
	{{end}}
	{{if and (ne .Recv "") (exported .Recv)}}
	  "{{.Recv}}"->f{{$i}}; 
	{{- end}}
    {{range $rt, $_ := .ReturnTypes }}
	  f{{$i}}->"{{$rt}}";
	{{end}}
  {{end}}
}`))
)

func (i Info) Render(w io.Writer) error {
	return dotTpl.Execute(w, i)
}

func constr(s string) bool {
	return strings.HasPrefix(s, "New") || strings.HasPrefix(s, "new")
}

// NewInfo returns a new template rendering type.
func NewInfo() *Info {
	return &Info{
		Types: map[string]bool{},
	}
}

func (d *Diagram) RecvType(r *ast.FieldList) string {
	if d.err != nil {
		return ""
	}
	if r == nil {
		return ""
	}
	recvType := r.List[0].Type
	return d.Sprint(recvType)
}

func (d *Diagram) FuncDecls(packages map[string]*ast.Package) *Info {
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

func (d *Diagram) render(decls *Info, w io.Writer) {
	if d.err != nil {
		return
	}
	d.err = decls.Render(w)
}

func (d *Diagram) Render() error {
	p := d.ParseDir(d.path)
	decls := d.FuncDecls(p)
	c := process.NewCommand(output)
	pipe := c.StdinPipe()
	c.Run()
	d.render(decls, pipe)
	d.err = c.Wait()
	return d.err
}
