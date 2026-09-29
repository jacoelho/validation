package validation_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	v "github.com/jacoelho/validation/v2"
)

func TestAT_QUALITY_RELEASE_007_MigrationContract(t *testing.T) {
	var typed v.Rule[int] = func(int) error { return nil }
	if typed(0) != nil {
		t.Fatal("Rule result contract changed")
	}
	type record struct{ Values []int }
	rule := v.Struct(
		v.Each[[]int](v.Min(1)).Field("values", func(value record) []int { return value.Values }),
		v.Check(func(value record) bool { return len(value.Values) > 0 }, func(record) error { return v.NewViolation("nonempty_record", nil) }),
	)
	issues := v.Issues(rule(record{Values: []int{0, 0}}))
	if len(issues) != 2 || issues[0].Code != v.CodeMin || issues[1].Code != v.CodeMin ||
		v.FormatPath(issues[0].Path) != "$.values[0]" || v.FormatPath(issues[1].Path) != "$.values[1]" {
		t.Fatalf("typed struct/field/collection migration result = %+v", issues)
	}
	issues = v.Issues(rule(record{}))
	if len(issues) != 1 || issues[0].Code != "nonempty_record" {
		t.Fatalf("all-error struct engine result = %+v", issues)
	}

	forbidden := map[string]bool{
		"Error": true, "Errors": true, "StructValidator": true, "StructField": true,
		"SliceField": true, "MapField": true, "SliceRule": true, "MapRule": true,
		"MapEntryRule": true, "Fatal": true, "RuleStopOnError": true,
		"Or": true, "RuleNot": true,
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			switch item := declaration.(type) {
			case *ast.FuncDecl:
				if item.Name.Name == "Error" && item.Recv != nil {
					continue
				}
				if forbidden[item.Name.Name] || strings.Contains(item.Name.Name, "Regex") || strings.Contains(item.Name.Name, "Regexp") {
					t.Errorf("legacy public function remains: %s", item.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range item.Specs {
					if named, ok := spec.(*ast.TypeSpec); ok && forbidden[named.Name.Name] {
						t.Errorf("legacy public type remains: %s", named.Name.Name)
					}
				}
			}
		}
	}
	guide, err := os.ReadFile("MIGRATION.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, guidance := range []string{
		"Rule[T] func(T) error", "All independent rules run", "RequiredPtr", "RequiredValue",
		"MapRequiredKey", "Issue.Path", "FormatPath", "named", "concrete nil checks",
		"RuleStopOnError", "RuleNot", "Regex constructor", "legacy pointer and error-slice",
	} {
		if !strings.Contains(strings.ToLower(string(guide)), strings.ToLower(guidance)) {
			t.Errorf("migration guide lacks %q", guidance)
		}
	}
}

func TestAT_QUALITY_RELEASE_009_FullGatePolicy(t *testing.T) {
	for _, checker := range []string{"tools/check_spec.py", "tools/check_acceptance.py"} {
		cmd := exec.Command("python3", checker)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s rejected the release scope: %v\n%s", checker, err, out)
		}
	}
	cmd := exec.Command("python3", "-m", "unittest", "tools.test_record_acceptance")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("release gate policy tests failed: %v\n%s", err, out)
	}
	for _, document := range []string{"README.md", "MIGRATION.md"} {
		data, err := os.ReadFile(document)
		if err != nil {
			t.Fatal(err)
		}
		blocks := fencedGoBlocks(string(data))
		if len(blocks) == 0 {
			t.Errorf("%s contains no Go example", document)
			continue
		}
		for index, block := range blocks {
			name := document + "/example"
			if len(blocks) > 1 {
				name += string(rune('0' + index))
			}
			t.Run(name, func(t *testing.T) {
				compilePublishedExample(t, document, block)
			})
		}
	}
}

func fencedGoBlocks(document string) []string {
	var blocks []string
	for rest := document; ; {
		opening := strings.Index(rest, "```go\n")
		if opening < 0 {
			return blocks
		}
		rest = rest[opening+len("```go\n"):]
		closing := strings.Index(rest, "\n```")
		if closing < 0 {
			return blocks
		}
		blocks = append(blocks, rest[:closing])
		rest = rest[closing+len("\n```"):]
	}
}

func compilePublishedExample(t *testing.T, document, source string) {
	t.Helper()
	if document == "MIGRATION.md" {
		source = "package main\nimport \"errors\"\n" + source + "\nfunc main() {}\n"
	}
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	module := "module publishedexample\n\ngo 1.27\n\nrequire github.com/jacoelho/validation/v2 v2.0.0\nreplace github.com/jacoelho/validation/v2 => " + root + "\n"
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(directory, "example"), ".")
	cmd.Dir = directory
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s example does not compile: %v\n%s", document, err, out)
	}
}

func TestAT_QUALITY_RELEASE_003_FuzzSeedOracles(t *testing.T) {
	cmd := exec.Command("go", "test", "-json", "-run", "^Fuzz(Paths|NumericBounds|SliceOccurrences|MapIssueOrder)$", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fuzz seed oracles failed: %v\n%s", err, out)
	}
	want := map[string]bool{
		"FuzzPaths":            false,
		"FuzzNumericBounds":    false,
		"FuzzSliceOccurrences": false,
		"FuzzMapIssueOrder":    false,
	}
	seeds := make(map[string]int, len(want))
	decoder := json.NewDecoder(bytes.NewReader(out))
	for decoder.More() {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if err := decoder.Decode(&event); err != nil {
			t.Fatalf("decode fuzz event: %v", err)
		}
		if _, exists := want[event.Test]; exists && event.Action == "pass" {
			want[event.Test] = true
		}
		for name := range want {
			if strings.HasPrefix(event.Test, name+"/seed#") && event.Action == "pass" {
				seeds[name]++
			}
		}
	}
	for name, passed := range want {
		if !passed || seeds[name] == 0 {
			t.Errorf("%s: passed=%t seed cases=%d", name, passed, seeds[name])
		}
	}
}

func TestAT_QUALITY_RELEASE_002_NativeMatrixPolicy(t *testing.T) {
	cmd := exec.Command("python3", "-m", "unittest", "tools.test_record_acceptance")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native matrix policy tests failed: %v\n%s", err, out)
	}
}

type acceptanceReport struct {
	Schema      string         `json:"schema"`
	Source      map[string]any `json:"source"`
	Repository  map[string]any `json:"repository"`
	Environment map[string]any `json:"environment"`
	Bindings    struct {
		Unbound int `json:"unbound"`
	} `json:"bindings"`
	Cases []struct {
		Result string `json:"result"`
	} `json:"cases"`
	Checks map[string]any `json:"checks"`
	Matrix struct {
		Rejected []struct {
			Reason string `json:"reason"`
		} `json:"rejected"`
	} `json:"matrix"`
	ReleaseAcceptance struct {
		Complete bool     `json:"complete"`
		Reasons  []string `json:"reasons"`
	} `json:"release_acceptance"`
}

func runAcceptanceReport(t *testing.T, mode, manifest string, matrixRecords ...string) acceptanceReport {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	args := []string{"tools/record_acceptance.py", mode, "--output", path}
	if manifest != "" {
		args = append(args, "--manifest", manifest)
	}
	for _, record := range matrixRecords {
		args = append(args, "--matrix-record", record)
	}
	cmd := exec.Command("python3", args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%s unexpectedly accepted release: %s", mode, out)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("acceptance report missing after %v: %v\n%s", err, readErr, out)
	}
	var report acceptanceReport
	if decodeErr := json.Unmarshal(data, &report); decodeErr != nil {
		t.Fatalf("decode acceptance report: %v", decodeErr)
	}
	return report
}

func writeMatrixRecord(t *testing.T, report acceptanceReport, alter func(map[string]any)) string {
	t.Helper()
	record := map[string]any{
		"schema":     report.Schema,
		"source":     report.Source,
		"repository": map[string]any{"commit": report.Repository["commit"]},
		"environment": map[string]any{
			"go_version": "go1.27.0",
			"goos":       "linux",
			"goarch":     "amd64",
			"is_native":  true,
		},
		"checks": map[string]any{"local_complete": true},
	}
	alter(record)
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "matrix.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAT_QUALITY_RELEASE_004_SpecTraceability(t *testing.T) {
	cmd := exec.Command("python3", "tools/check_spec.py")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("specification traceability failed: %v\n%s", err, out)
	}
}

func TestAT_QUALITY_RELEASE_005_UnboundCaseBlocksAcceptance(t *testing.T) {
	cmd := exec.Command("python3", "tools/check_spec.py")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("specification structure failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile("acceptance/bindings.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	cases := manifest["cases"].([]any)
	for _, raw := range cases {
		entry := raw.(map[string]any)
		if entry["status"] == "bound" {
			entry["status"] = "unbound"
			entry["reason"] = "synthetic missing implementation binding"
			delete(entry, "test")
			delete(entry, "command")
			delete(entry, "evidence")
			break
		}
	}
	data, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "bindings.json")
	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	report := runAcceptanceReport(t, "--dry-run", manifestPath)
	if report.Bindings.Unbound == 0 {
		t.Fatal("fixture no longer has an unbound case")
	}
	found := false
	for _, result := range report.Cases {
		found = found || result.Result == "unbound"
	}
	if !found || report.ReleaseAcceptance.Complete {
		t.Fatalf("unbound case did not block release: found=%t complete=%t reasons=%v", found, report.ReleaseAcceptance.Complete, report.ReleaseAcceptance.Reasons)
	}
	if !containsReason(report.ReleaseAcceptance.Reasons, "unbound") {
		t.Fatalf("release did not explain unbound case: %v", report.ReleaseAcceptance.Reasons)
	}
}

func TestAT_QUALITY_RELEASE_006_RejectsStaleEvidence(t *testing.T) {
	base := runAcceptanceReport(t, "--dry-run", "")
	oldCommit := writeMatrixRecord(t, base, func(record map[string]any) {
		record["repository"].(map[string]any)["commit"] = "0000000000000000000000000000000000000000"
	})
	oldFeature := writeMatrixRecord(t, base, func(record map[string]any) {
		source := make(map[string]any, len(base.Source))
		for key, value := range base.Source {
			source[key] = value
		}
		source["feature_sha256"] = map[string]string{"features/01_core.feature": "wrong"}
		record["source"] = source
	})
	report := runAcceptanceReport(t, "--aggregate-only", "", oldCommit, oldFeature)
	var rejectedCommit, rejectedFeature bool
	for _, item := range report.Matrix.Rejected {
		rejectedCommit = rejectedCommit || item.Reason == "implementation commit differs"
		rejectedFeature = rejectedFeature || item.Reason == "specification or feature hash differs"
	}
	if !rejectedCommit || !rejectedFeature || report.ReleaseAcceptance.Complete {
		t.Fatalf("stale evidence accepted: commit=%t feature=%t complete=%t rejected=%+v", rejectedCommit, rejectedFeature, report.ReleaseAcceptance.Complete, report.Matrix.Rejected)
	}
}

func containsReason(reasons []string, part string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, part) {
			return true
		}
	}
	return false
}
