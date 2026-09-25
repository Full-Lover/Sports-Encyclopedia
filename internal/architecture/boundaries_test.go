package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestModuleImportBoundaries(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	for _, module := range []string{"atlasregistry", "publishedatlas", "webexperience"} {
		directory := filepath.Join(root, module)
		err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range file.Imports {
				name, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				if forbiddenModuleImport(module, name) {
					t.Errorf("%s imports forbidden dependency %s", path, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func forbiddenModuleImport(module, imported string) bool {
	switch module {
	case "atlasregistry":
		return imported == "net/http" || strings.Contains(imported, "/internal/publishedatlas") ||
			strings.Contains(imported, "/internal/webexperience")
	case "publishedatlas":
		return strings.Contains(imported, "/internal/atlasregistry") ||
			strings.Contains(imported, "/internal/webexperience")
	case "webexperience":
		return imported == "database/sql" || strings.HasPrefix(imported, "github.com/go-sql-driver/mysql") ||
			strings.Contains(imported, "/internal/atlasregistry")
	default:
		return false
	}
}

func TestForbiddenModuleImportRules(t *testing.T) {
	for _, sample := range []struct {
		module, dependency string
		forbidden          bool
	}{
		{"atlasregistry", "net/http", true},
		{"atlasregistry", "github.com/full-lover/sports-encyclopedia/internal/publishedatlas", true},
		{"atlasregistry", "database/sql", false},
		{"publishedatlas", "github.com/full-lover/sports-encyclopedia/internal/atlasregistry", true},
		{"webexperience", "database/sql", true},
		{"webexperience", "github.com/go-sql-driver/mysql", true},
		{"webexperience", "github.com/full-lover/sports-encyclopedia/internal/publishedatlas", false},
	} {
		if got := forbiddenModuleImport(sample.module, sample.dependency); got != sample.forbidden {
			t.Fatalf("%s importing %s: forbidden = %t, want %t",
				sample.module, sample.dependency, got, sample.forbidden)
		}
	}
}
