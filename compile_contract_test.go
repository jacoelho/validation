package validation_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileTypeContract(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	line := strings.SplitN(string(mod), "\n", 2)[0]
	module := strings.TrimPrefix(line, "module ")
	version := "v2.0.0"
	fixtures := []struct {
		name, source string
		compiles     bool
	}{
		{"named types", `package fixture
import v "MODULE"
type Age int64
type Tags []string
type Scores map[string]int
var _ = v.Min(Age(18))
var _ = v.SliceMinLength[Tags](1)
var _ = v.MapLength[Scores](1)
var _ = v.Field("age",func(a struct{Age Age})Age{return a.Age},v.Min(Age(0)))
`, true},
		{"string is not numeric", `package fixture
import v "MODULE"
var _ = v.Min("abc")
`, false},
		{"complex is not numeric", `package fixture
import v "MODULE"
var _ = v.Min(complex(1,2))
`, false},
		{"integer is not string", `package fixture
import v "MODULE"
var _ = v.RuneLength[int](2)
`, false},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			dir := t.TempDir()
			goMod := fmt.Sprintf("module fixture\n\ngo 1.27\n\nrequire %s %s\nreplace %s => %s\n", module, version, module, root)
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0600); err != nil {
				t.Fatal(err)
			}
			source := strings.ReplaceAll(fixture.source, "MODULE", module)
			if err := os.WriteFile(filepath.Join(dir, "fixture_test.go"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "test", "./...")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=off")
			out, err := cmd.CombinedOutput()
			if fixture.compiles && err != nil {
				t.Fatalf("positive fixture failed: %v\n%s", err, out)
			}
			if !fixture.compiles && err == nil {
				t.Fatalf("negative fixture compiled unexpectedly:\n%s", out)
			}
		})
	}
}
