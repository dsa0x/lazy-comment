package lazycomment

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
)

func main() {

	c := flag.String("c", "--default comment--", "add default comment")
	dir := flag.String("dir", "--", "directory")
	flag.Parse()
	if *dir == "" {
		err := errors.New("directory or file path required. -dir")
		cleanExit(err)
	} else {
		fi, err := os.Stat(*dir)
		if err != nil {
			cleanExit(err)
		}
		if fi.IsDir() {
			err = LazyCommenter(*dir, *c, true)
		} else {
			err = LazyCommenter(*dir, *c, false)
		}
		if err != nil {
			cleanExit(err)
		}
	}

}

type Visitor struct {
	visits map[string]int
}

func (visitor Visitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return visitor
	}
	updatedVisits := Visit(node, visitor.visits)
	visitor.visits = updatedVisits
	return visitor
}

func LazyCommenter(dir string, comment string, directory bool) (err error) {
	fset := token.NewFileSet()
	pkgs := make(map[string]*ast.Package)
	if directory {
		pkgs, err = parser.ParseDir(fset, dir, nil, parser.ParseComments)
		if err != nil {
			return err
		}
	} else {
		file, err := parser.ParseFile(fset, dir, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		names := strings.Split(dir, "/")
		fileName := names[len(names)-1]
		pkgs[file.Name.Name] = &ast.Package{
			Name:  fileName,
			Scope: file.Scope,
			Files: map[string]*ast.File{fileName: file},
		}
	}

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			comments := []*ast.CommentGroup{}

			visitor := Visitor{visits: make(map[string]int)}
			ast.Inspect(file, func(n ast.Node) bool {

				if n == nil {
					return true
				}
				c, ok := n.(*ast.CommentGroup)
				if ok {
					comments = append(comments, c)
				}
				ast.Walk(visitor, n)
				FindComment(n, file, visitor.visits, comment)
				return true
			})

			file.Comments = comments

			buf := new(bytes.Buffer)
			err := format.Node(buf, fset, file)
			fmt.Println(fileName)
			if err != nil {
				fmt.Printf("lazy comment error: %v\n", err)
			} else {
				ioutil.WriteFile(fileName, buf.Bytes(), 0664)
			}
		}
	}
	return
}

func FindComment(node ast.Node, file *ast.File, visits map[string]int, defaultComment string) {

	switch node.(type) {
	case *ast.FuncDecl:
		fn, _ := node.(*ast.FuncDecl)
		fmt.Println(fn.Name.Name, fn.Type.Params)
		if fn.Name.IsExported() && fn.Doc.Text() == "" {
			comment := &ast.Comment{
				Text:  fmt.Sprintf("// %s %s", fn.Name.Name, defaultComment),
				Slash: fn.Pos() - 1,
			}
			cg := &ast.CommentGroup{
				List: []*ast.Comment{comment},
			}
			fn.Doc = cg
		}

		key := fmt.Sprintf("%s%d", fn.Name.Name, fn.Pos())
		if visits[key] > 2 {
			fn.Doc = &ast.CommentGroup{}
		}

	case *ast.GenDecl:
		gn, _ := node.(*ast.GenDecl)

		for _, v := range gn.Specs {
			spc, ok := v.(*ast.TypeSpec)
			if ok && spc.Name.IsExported() && gn.Doc.Text() == "" {
				comment := &ast.Comment{
					Text:  fmt.Sprintf("//%s %s", spc.Name.Name, defaultComment),
					Slash: gn.Pos() - 1,
				}
				cg := &ast.CommentGroup{
					List: []*ast.Comment{comment},
				}
				spc.Doc = cg
			}

			if ok && visits[spc.Name.Name] > 2 {
				spc.Doc = &ast.CommentGroup{}
			}

			if vpc, ok := v.(*ast.ValueSpec); ok && gn.Doc.Text() == "" {
				comment := &ast.Comment{
					Text:  fmt.Sprintf("// %s %s", vpc.Names[0].Name, defaultComment),
					Slash: gn.Pos() - 1,
				}
				cg := &ast.CommentGroup{
					List: []*ast.Comment{comment},
				}
				vpc.Doc = cg

				key := fmt.Sprintf("%s%d", vpc.Names[0].Name, gn.Pos())
				if visits[key] > 2 {
					vpc.Doc = &ast.CommentGroup{}
				}
			}
		}

	}
}

func Visit(n ast.Node, visits map[string]int) map[string]int {

	if n == nil {
		return visits
	}

	switch d := n.(type) {
	case *ast.FuncDecl:
		key := fmt.Sprintf("%s%d", d.Name.Name, d.Pos())
		visits[key]++
		return visits
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			if value, ok := spec.(*ast.ValueSpec); ok {
				for _, name := range value.Names {
					key := fmt.Sprintf("%s%d", name.Name, d.Pos())
					visits[key]++
					return visits
				}
			}
		}
	}
	return visits
}

func cleanExit(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
