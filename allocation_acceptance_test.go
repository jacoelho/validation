//go:build !race

package validation_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestAT_ALLOCATION_006_UniquenessTradeoff(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "slices.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var unique *ast.FuncDecl
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "SliceUnique" {
			unique = function
			break
		}
	}
	if unique == nil {
		t.Fatal("SliceUnique source not found")
	}
	var rule *ast.FuncLit
	ast.Inspect(unique.Body, func(node ast.Node) bool {
		if literal, ok := node.(*ast.FuncLit); ok && rule == nil {
			rule = literal
		}
		return true
	})
	if rule == nil || rule.Type.Params == nil || len(rule.Type.Params.List) != 1 || len(rule.Type.Params.List[0].Names) != 1 {
		t.Fatal("SliceUnique has no single-input rule")
	}
	input := rule.Type.Params.List[0].Names[0].Name
	previousScan := false
	ast.Inspect(rule.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.MapType:
			t.Error("SliceUnique uses a per-call map")
		case *ast.RangeStmt:
			if isSourceName(n.X, input) {
				index, ok := n.Key.(*ast.Ident)
				if ok {
					ast.Inspect(n.Body, func(child ast.Node) bool {
						loop, ok := child.(*ast.ForStmt)
						if !ok {
							return true
						}
						condition, ok := loop.Cond.(*ast.BinaryExpr)
						if !ok || condition.Op != token.LSS || !isSourceName(condition.Y, index.Name) {
							return true
						}
						prior, ok := condition.X.(*ast.Ident)
						if !ok {
							return true
						}
						ast.Inspect(loop.Body, func(expression ast.Node) bool {
							comparison, ok := expression.(*ast.BinaryExpr)
							if ok && comparison.Op == token.EQL &&
								(sourceIndex(comparison.X, input, prior.Name) && sourceIndex(comparison.Y, input, index.Name) ||
									sourceIndex(comparison.Y, input, prior.Name) && sourceIndex(comparison.X, input, index.Name)) {
								previousScan = true
							}
							return true
						})
						return true
					})
				}
			}
		case *ast.AssignStmt:
			for _, left := range n.Lhs {
				if indexed, ok := left.(*ast.IndexExpr); ok && isSourceName(indexed.X, input) {
					t.Error("SliceUnique mutates input")
				}
			}
		}
		return true
	})
	if !previousScan {
		t.Fatal("SliceUnique lacks previous-element equality scan")
	}
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "n(n−1)/2") {
		t.Fatal("README lacks the documented worst-case comparison count")
	}

	cmd := exec.Command("go", "test", "-run", "^$", "-bench", "^BenchmarkSliceUnique$", "-benchmem", "-benchtime=1x", "-count=1", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("uniqueness benchmark failed: %v\n%s", err, out)
	}
	for _, size := range []string{"8", "64", "1024"} {
		pattern := regexp.MustCompile(`(?m)^BenchmarkSliceUnique/` + size + `-\d+\s+\d+\s+\S+ ns/op\s+` + size + `(?:\.0+)? input-items\s+\d+ B/op\s+\d+ allocs/op$`)
		if !pattern.Match(out) {
			t.Errorf("missing uniqueness benchmark row for size %s:\n%s", size, out)
		}
	}
	if !strings.Contains(string(out), "compiler="+runtime.Version()) || !strings.Contains(string(out), "platform="+runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("uniqueness benchmark lacks compiler/platform metadata:\n%s", out)
	}
	report := nativeAllocationEvidence(t)
	for _, size := range []string{"8", "64", "1024"} {
		name := "slice-size-" + size
		found := false
		for _, fixture := range report.Fixtures {
			if fixture.Name != name {
				continue
			}
			found = true
			if fixture.Kind != "allocation-free" {
				t.Errorf("%s has kind %q", name, fixture.Kind)
			}
			for _, phase := range []string{"first", "batch"} {
				probe, ok := fixture.Phases[phase]
				if !ok || probe.Status != "passed" || probe.Mallocs != 0 || probe.Bytes != 0 {
					t.Errorf("%s/%s allocation = %+v", name, phase, probe)
				}
			}
		}
		if !found {
			t.Errorf("missing uniqueness allocation fixture %s", name)
		}
	}
}

func isSourceName(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

func sourceIndex(expression ast.Expr, slice, index string) bool {
	indexed, ok := expression.(*ast.IndexExpr)
	return ok && isSourceName(indexed.X, slice) && isSourceName(indexed.Index, index)
}

type allocationEvidence struct {
	Schema   string   `json:"schema"`
	Status   string   `json:"status"`
	Failures []string `json:"failures"`
	Metadata struct {
		Compilers []string `json:"compilers"`
		Platforms []string `json:"platforms"`
	} `json:"metadata"`
	Fixtures []struct {
		Name   string `json:"name"`
		Kind   string `json:"kind"`
		Phases map[string]struct {
			Status  string `json:"status"`
			Calls   int    `json:"calls"`
			Mallocs uint64 `json:"mallocs"`
			Bytes   uint64 `json:"bytes"`
			Go      string `json:"go"`
			OS      string `json:"os"`
			Arch    string `json:"arch"`
		} `json:"phases"`
	} `json:"fixtures"`
}

func TestAT_ALLOCATION_NativeProbes(t *testing.T) {
	report := nativeAllocationEvidence(t)
	byName := make(map[string]int, len(report.Fixtures))
	for index, fixture := range report.Fixtures {
		if _, duplicate := byName[fixture.Name]; duplicate {
			t.Fatalf("duplicate allocation fixture %q", fixture.Name)
		}
		byName[fixture.Name] = index
	}
	rows := [][]string{
		{"scalar-signed", "scalar-unsigned-named", "scalar-float", "scalar-comparable"},
		{"strings-named", "bytes-named"},
		{"time"},
		{"fields-flat-16"},
		{"fields-deep-16"},
		{"presence-required-present", "presence-nested-pointer"},
		{"presence-optional-absent"},
		{"presence-value-absent", "presence-value-present"},
		{"slice-size-0", "slice-size-1", "slice-size-8", "slice-size-64", "slice-size-1024"},
		{"slice-nested-named"},
		{"map-size-0", "map-size-1", "map-size-8", "map-size-64", "map-size-1024", "map-size-10000"},
		{"membership-size-0", "membership-size-1", "membership-size-4", "membership-size-32", "membership-size-1024"},
		{"slice-size-0", "slice-size-1", "slice-size-8", "slice-size-64", "slice-size-1024"},
		{"array-pointer-project"},
		{"check-last-alternative"},
		{"scalar-comparator"},
	}
	for row, names := range rows {
		for _, name := range names {
			index, ok := byName[name]
			if !ok {
				t.Errorf("allocation row %d missing fixture %q", row+1, name)
				continue
			}
			fixture := report.Fixtures[index]
			if fixture.Kind != "allocation-free" {
				t.Errorf("allocation row %d fixture %q kind=%q", row+1, name, fixture.Kind)
			}
			for phase, calls := range map[string]int{"first": 1, "batch": 1000} {
				probe, ok := fixture.Phases[phase]
				if !ok || probe.Status != "passed" || probe.Calls != calls || probe.Mallocs != 0 || probe.Bytes != 0 ||
					probe.Go != runtime.Version() || probe.OS != runtime.GOOS || probe.Arch != runtime.GOARCH {
					t.Errorf("allocation row %d fixture %q phase %q = %+v", row+1, name, phase, probe)
				}
			}
		}
	}
	for _, control := range []struct {
		name          string
		firstPositive bool
		batchPositive bool
	}{
		{"control-blank", false, false},
		{"control-first", true, true},
		{"control-rare", false, true},
		{"control-callback", true, true},
	} {
		index, ok := byName[control.name]
		if !ok {
			t.Errorf("missing allocation control %q", control.name)
			continue
		}
		fixture := report.Fixtures[index]
		if fixture.Kind != "control" {
			t.Errorf("allocation control %q kind=%q", control.name, fixture.Kind)
		}
		for _, phase := range []struct {
			name     string
			positive bool
		}{{"first", control.firstPositive}, {"batch", control.batchPositive}} {
			probe, ok := fixture.Phases[phase.name]
			positive := probe.Mallocs > 0 && probe.Bytes > 0
			zero := probe.Mallocs == 0 && probe.Bytes == 0
			if !ok || probe.Status != "passed" || (phase.positive && !positive) || (!phase.positive && !zero) {
				t.Errorf("allocation control %q phase %q = %+v", control.name, phase.name, probe)
			}
		}
	}
}

func nativeAllocationEvidence(t *testing.T) allocationEvidence {
	t.Helper()
	path := filepath.Join(t.TempDir(), "allocation.json")
	cmd := exec.Command("python3", "tools/check_allocation.py", "--json-output", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native allocation gate failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("allocation gate did not write evidence: %v", err)
	}
	var report allocationEvidence
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode allocation evidence: %v", err)
	}
	if report.Schema != "validation.allocation-evidence.v1" || report.Status != "passed" || len(report.Failures) != 0 {
		t.Fatalf("allocation gate status: schema=%q status=%q failures=%v", report.Schema, report.Status, report.Failures)
	}
	if len(report.Metadata.Compilers) != 1 || report.Metadata.Compilers[0] != runtime.Version() ||
		len(report.Metadata.Platforms) != 1 || report.Metadata.Platforms[0] != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("allocation evidence toolchain/platform = %+v, want %s %s/%s", report.Metadata, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	}
	return report
}

func TestAT_ALLOCATION_004_BenchmarkRows(t *testing.T) {
	cmd := exec.Command("go", "test", "-run", "^$", "-bench", ".", "-benchmem", "-benchtime=1x", "-count=1", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("benchmark smoke failed: %v\n%s", err, out)
	}
	text := string(out)
	required := []string{
		"BenchmarkConstruction/",
		"BenchmarkValidation/record-fields-3-tags-2/valid-",
		"BenchmarkValidation/record-fields-3-tags-2/invalid-",
		"BenchmarkValidation/record-fields-3-tags-2/reporting-",
		"BenchmarkTraversal/",
	}
	for _, name := range required {
		if !strings.Contains(text, name) {
			t.Errorf("benchmark output lacks %q:\n%s", name, out)
		}
	}
	row := regexp.MustCompile(`(?m)^Benchmark\S+\s+\d+\s+\S+ ns/op\s+\S+ input-items\s+\d+ B/op\s+\d+ allocs/op$`)
	rows := row.FindAllString(text, -1)
	if len(rows) < 8 {
		t.Fatalf("benchmark output has %d complete metric rows, want at least 8:\n%s", len(rows), out)
	}
	if !strings.Contains(text, "compiler="+runtime.Version()) || !strings.Contains(text, "platform="+runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("benchmark output lacks exact compiler/platform metadata:\n%s", out)
	}
}
