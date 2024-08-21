package strcase

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"testing"
)

func parseFuncs(t *testing.T, filename string) []string {
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range af.Decls {
		if fd, _ := d.(*ast.FuncDecl); fd != nil {
			if fd.Name == nil {
				continue
			}
			name := fd.Name.Name
			if ast.IsExported(name) {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// Test that the strcase and bytcase packages have the same API
func TestPackageParity(t *testing.T) {
	strnames := parseFuncs(t, "strcase.go")
	bytenames := parseFuncs(t, "bytcase/bytcase.go")
	if !reflect.DeepEqual(strnames, bytenames) {
		t.Fatalf("The API of the strcase and bytcase packages differs:\n"+
			"strcase: %q\n"+
			"bytcase: %q\n", strnames, bytenames)
	}
}
