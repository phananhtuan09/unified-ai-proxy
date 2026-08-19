package architecture

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionImportDirection(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "list", "-json", "./internal/...")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	imports := map[string][]string{}
	for {
		var p struct {
			ImportPath string
			Imports    []string
		}
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		imports[p.ImportPath] = p.Imports
	}
	for importer, deps := range imports {
		for _, dep := range deps {
			if forbidden(importer, dep) {
				t.Errorf("architecture violation: %s imports %s", importer, dep)
			}
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() && forbiddenPackage(entry.Name()) {
			t.Errorf("forbidden generic package: internal/%s", entry.Name())
		}
	}
}

func forbidden(importer, imported string) bool {
	if !strings.Contains(importer, "/internal/") {
		return false
	}
	for _, name := range []string{"server", "cli", "tui"} {
		if strings.HasSuffix(imported, "/internal/"+name) {
			return strings.HasSuffix(importer, "/internal/provider") || strings.HasSuffix(importer, "/internal/proxy") || strings.HasSuffix(importer, "/internal/model")
		}
	}
	return false
}

func forbiddenPackage(name string) bool {
	return map[string]bool{"utils": true, "common": true, "shared": true, "services": true, "handlers": true}[name]
}

func Example_forbidden() {
	fmt.Println(forbidden("example/internal/proxy", "example/internal/server"))
	// Output: true
}
