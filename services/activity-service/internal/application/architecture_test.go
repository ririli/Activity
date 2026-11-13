package application

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestBusinessLayersHaveNoAdapterDependencies(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	internal := filepath.Dir(filepath.Dir(file))
	err := filepath.WalkDir(internal, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		if !strings.HasSuffix(dir, "/application") && !strings.HasSuffix(dir, "/domain") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			v, _ := strconv.Unquote(imp.Path.Value)
			if v == "activity/pkg/platform" || v == "database/sql" || strings.Contains(v, "/infrastructure") || strings.Contains(v, "/interfaces") || strings.Contains(v, "gin-gonic") || strings.Contains(v, "redis") || strings.Contains(v, "amqp") {
				t.Errorf("business layer imports adapter %s: %s", path, v)
			}
			if strings.HasSuffix(dir, "/domain") && strings.Contains(v, "/application") {
				t.Errorf("domain imports application: %s", path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			field, ok := n.(*ast.Field)
			if ok && field.Tag != nil && strings.Contains(field.Tag.Value, "binding:") {
				t.Errorf("HTTP binding leaked into %s", path)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
