package validation_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeSourcePolicy(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := make([]*ast.File, 0)
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatal("no production Go files inspected")
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	config := types.Config{Importer: importer.Default(), GoVersion: "go1.27"}
	if _, err := config.Check("github.com/jacoelho/validation/v2", fset, files, info); err != nil {
		t.Fatalf("runtime source type check: %v", err)
	}
	for _, violation := range sourcePolicyViolations(files, fset, info) {
		t.Error(violation)
	}
	deps, err := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("runtime dependency graph: %v\n%s", err, deps)
	}
	paths := strings.Fields(string(deps))
	if len(paths) == 0 {
		t.Fatal("runtime dependency graph has no module package")
	}
	for _, path := range paths {
		if path != "github.com/jacoelho/validation/v2" {
			t.Errorf("non-standard runtime dependency: %s", path)
		}
	}
}

// TestATTYPINGARCHITECTURE004SourcePolicyRejectsInputErasure proves the
// source gate's type-erasure distinctions with fixtures that are independent
// of the production implementation.
func TestATTYPINGARCHITECTURE004SourcePolicyRejectsInputErasure(t *testing.T) {
	files, fset, info := parseSourceFixture(t, `package fixture

func genericConstraint[T any](value T) T { return value }

func walkIssueNode(err error) {
	_, _ = err.(interface{ Unwrap() error })
}

func validate[T any](input T) error {
	_ = any(input)
	var value interface{} = input
	switch value.(type) {
	case int:
	}
	var erased any
	_, _ = erased.(int)
	var err error
	_, _ = err.(interface{ Unwrap() error })
	return nil
}
`, true)
	violations := sourcePolicyViolations(files, fset, info)
	requireSourceViolation(t, violations, "conversion to empty interface")
	requireSourceViolation(t, violations, "runtime value type switch")
	requireSourceViolation(t, violations, "value type assertion")
	requireSourceViolation(t, violations, "error Unwrap assertion outside failure-side inspection")

	files, fset, info = parseSourceFixture(t, `package fixture

func genericConstraint[T any](value T) T { return value }

func walkIssueNode(err error) {
	_, _ = err.(interface{ Unwrap() error })
}
`, true)
	if violations := sourcePolicyViolations(files, fset, info); len(violations) != 0 {
		t.Fatalf("valid type-constraint and failure-side fixture rejected: %v", violations)
	}
}

// TestATTYPINGARCHITECTURE005SourcePolicyRejectsScopeDependencies exercises
// the import boundary independently from the production package's dependency
// graph.
func TestATTYPINGARCHITECTURE005SourcePolicyRejectsScopeDependencies(t *testing.T) {
	files, fset, _ := parseSourceFixture(t, `package fixture

import (
	_ "context"
	_ "regexp"
	_ "os"
	_ "github.com/cucumber/godog"
)
`, false)
	violations := sourcePolicyViolations(files, fset, nil)
	for _, forbidden := range []string{`"context"`, `"regexp"`, `"os"`, `"github.com/cucumber/godog"`} {
		requireSourceViolation(t, violations, "forbidden runtime import "+forbidden)
	}

	files, fset, _ = parseSourceFixture(t, `package fixture

import (
	_ "math"
	_ "time"
)
`, false)
	if violations := sourcePolicyViolations(files, fset, nil); len(violations) != 0 {
		t.Fatalf("allowed standard-library fixture rejected: %v", violations)
	}

	files, fset, _ = parseSourceFixture(t, `package fixture

import "fmt"

func implicitOutput(value string) {
	fmt.Println(value)
}
`, false)
	requireSourceViolation(t, sourcePolicyViolations(files, fset, nil), "implicit I/O call")

	files, fset, _ = parseSourceFixture(t, `package fixture

func Regex(value string) bool { return true }
func ParseTags(value string) bool { return true }
func FormatEmail(value string) bool { return true }
func WithContext(value string) bool { return true }
type TagEngine struct{}
var RegexCache int
`, false)
	violations = sourcePolicyViolations(files, fset, nil)
	for _, name := range []string{"Regex", "ParseTags", "FormatEmail", "WithContext", "TagEngine", "RegexCache"} {
		requireSourceViolation(t, violations, "out-of-scope runtime export "+name)
	}
}

// TestATITERATION005SourcePolicySeparatesValidationAndReporting checks the
// direct-evaluation boundary with both named and structurally-typed reporter
// callbacks. It deliberately keeps the reporting call graph separate from
// validation so the checker does not depend on implementation line names.
func TestATITERATION005SourcePolicySeparatesValidationAndReporting(t *testing.T) {
	files, fset, info := parseSourceFixture(t, `package fixture

import "iter"

type Collector func(error) bool

func validate(value int, collect Collector) error {
	var channel chan error
	_ = channel
	seq := iter.Seq[error](func(yield func(error) bool) {})
	next, stop := iter.Pull(seq)
	_ = next
	stop()
	_ = walkIssues(nil)
	_ = Issues(nil)
	_ = Format(nil)
	_ = collect
	return nil
}

func Issues(error) []error { return nil }
func Format(error) string { return "" }

func walkIssues(err error) iter.Seq[error] {
	return func(yield func(error) bool) {}
}
`, true)
	violations := sourcePolicyViolations(files, fset, info)
	requireSourceViolation(t, violations, "runtime channel type")
	requireSourceViolation(t, violations, "iter.Pull")
	requireSourceViolation(t, violations, "iter.Seq outside failure-side inspection")
	requireSourceViolation(t, violations, "reporting iterator called from validation function")
	requireSourceViolation(t, violations, "reporting function called from validation function")
	requireSourceViolation(t, violations, "reporter callback in validation ABI")

	files, fset, info = parseSourceFixture(t, `package fixture

import "iter"

func Issues(err error) {
	for range walkIssues(err) {
	}
}

func walkIssues(err error) iter.Seq[error] {
	return func(yield func(error) bool) {}
}
`, true)
	if violations := sourcePolicyViolations(files, fset, info); len(violations) != 0 {
		t.Fatalf("reporting-only iterator fixture rejected: %v", violations)
	}
}

// TestATALLOCATION007SourcePolicyRejectsHiddenState catches pooled state,
// mutable state captured through a named closure, package state, and mutation
// methods. A read-only constructor lookup remains valid.
func TestATALLOCATION007SourcePolicyRejectsHiddenState(t *testing.T) {
	files, fset, info := parseSourceFixture(t, `package fixture

import "sync"

var pool sync.Pool

var globalScratch []int

func Rule() func(int) error {
	scratch := make([]int, 0, 1)
	rule := func(value int) error {
		scratch = append(scratch, value)
		globalScratch = append(globalScratch, value)
		return nil
	}
	return rule
}

func PoolRule(value int) error {
	_ = pool.Get()
	return nil
}

func MapRule() func(int) error {
	cache := make(map[int]int)
	return func(value int) error {
		cache[value] = value
		return nil
	}
}

func SyncMapRule() func(int) error {
	var cache sync.Map
	return func(value int) error {
		cache.Store(value, value)
		return nil
	}
}

func PerCallMapRule() func(int) error {
	return func(value int) error {
		cache := make(map[int]int)
		cache[value] = value
		return nil
	}
}
`, true)
	violations := sourcePolicyViolations(files, fset, info)
	requireSourceViolation(t, violations, "sync.Pool")
	requireSourceViolation(t, violations, "package-level mutable state")
	requireSourceViolation(t, violations, "mutable state captured by returned rule")
	requireSourceViolation(t, violations, "per-call map inside returned rule")

	files, fset, info = parseSourceFixture(t, `package fixture

func NamedRule() func(int) error {
	var scratch []int
	rule := func(value int) error {
		scratch = append(scratch, value)
		return nil
	}
	return rule
}
`, true)
	requireSourceViolation(t, sourcePolicyViolations(files, fset, info), "mutable state captured by returned rule")

	files, fset, info = parseSourceFixture(t, `package fixture

func ReadOnlyRule() func(int) error {
	allowed := map[int]struct{}{1: struct{}{}}
	return func(value int) error {
		_, _ = allowed[value]
		return nil
	}
}
`, true)
	if violations := sourcePolicyViolations(files, fset, info); len(violations) != 0 {
		t.Fatalf("read-only constructor state rejected: %v", violations)
	}
}

// TestATERRORS009SourcePolicyChecksMultiUnwrapOwnership verifies that a
// library-owned multi-error node exposes a copy of its child slice. Cycles and
// malformed external trees remain outside the static gate by contract.
func TestATERRORS009SourcePolicyChecksMultiUnwrapOwnership(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("docs", "validation-refactor-spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"Library-generated error trees shall be finite and acyclic.",
		"Cyclic external error graphs and malformed multi-Unwrap nil children violate the extension contract",
		"Repeated nodes are visited once per occurrence, not once per pointer identity.",
	} {
		if !strings.Contains(string(contract), statement) {
			t.Fatalf("error-tree contract is missing %q", statement)
		}
	}

	files, fset, info := parseSourceFixture(t, `package fixture

type aggregate struct { children []error }

func (a aggregate) Error() string { return "aggregate" }
func (a aggregate) Unwrap() []error { return a.children }
`, true)
	violations := sourcePolicyViolations(files, fset, info)
	requireSourceViolation(t, violations, "multi-error Unwrap returns caller-owned slice")

	files, fset, info = parseSourceFixture(t, `package fixture

type aggregate struct { children []error }

func (a aggregate) Error() string { return "aggregate" }
func (a aggregate) Unwrap() []error { return append([]error(nil), a.children...) }
`, true)
	if violations := sourcePolicyViolations(files, fset, info); len(violations) != 0 {
		t.Fatalf("defensive multi-error Unwrap rejected: %v", violations)
	}

	files, fset, info = parseSourceFixture(t, `package fixture

type aggregate struct { children []error }

func (a aggregate) Error() string { return "aggregate" }

func combine(children []error) error {
	return &aggregate{children: children}
}

func appendFailure(children []error, err error) []error {
	return append(children, err)
}

func (a aggregate) Unwrap() []error { return append([]error(nil), a) }
`, true)
	violations = sourcePolicyViolations(files, fset, info)
	requireSourceViolation(t, violations, "aggregate stores unfiltered child slice")
	requireSourceViolation(t, violations, "error aggregate appends nil child")
	requireSourceViolation(t, violations, "multi-error Unwrap returns receiver")

	files, fset, info = parseSourceFixture(t, `package fixture

type aggregate struct{ children []error }

func (a aggregate) Error() string { return "aggregate" }
func (a aggregate) Unwrap() []error { return append([]error(nil), a.children...) }

func appendFailure(children []error, err error) []error {
	if err != nil {
		return append(children, err)
	}
	return children
}

func combine(children []error) error {
	return &aggregate{children: append([]error(nil), children...)}
}
`, true)
	if violations := sourcePolicyViolations(files, fset, info); len(violations) != 0 {
		t.Fatalf("well-formed error-tree fixture rejected: %v", violations)
	}
}

func parseSourceFixture(t *testing.T, source string, typeCheck bool) ([]*ast.File, *token.FileSet, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source fixture: %v", err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	if typeCheck {
		config := types.Config{Importer: importer.Default(), GoVersion: "go1.27"}
		if _, err := config.Check("fixture", fset, []*ast.File{file}, info); err != nil {
			t.Fatalf("type-check source fixture: %v", err)
		}
	}
	return []*ast.File{file}, fset, info
}

func requireSourceViolation(t *testing.T, violations []string, fragment string) {
	t.Helper()
	for _, violation := range violations {
		if strings.Contains(violation, fragment) {
			return
		}
	}
	t.Fatalf("source policy did not report %q; violations=%v", fragment, violations)
}

func sourcePolicyViolations(files []*ast.File, fset *token.FileSet, info *types.Info) []string {
	var violations []string
	add := func(file *ast.File, pos token.Pos, message string) {
		filename := fset.Position(file.Pos()).Filename
		if filename == "" {
			filename = "<source>"
		}
		violations = append(violations, filename+":"+fset.Position(pos).String()+": "+message)
	}
	for _, file := range files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !isAllowedRuntimeImport(path) {
				add(file, imp.Pos(), "forbidden runtime import "+strconvQuote(path))
			}
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if strings.HasPrefix(comment.Text, "//go:linkname") || strings.HasPrefix(comment.Text, "//go:nosplit") {
					add(file, comment.Pos(), "unsafe directive "+comment.Text)
				}
			}
		}
		iterQualifiers := importQualifiers(file, "iter")
		syncQualifiers := importQualifiers(file, "sync")
		fmtQualifiers := importQualifiers(file, "fmt")
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.TypeSwitchStmt:
				add(file, n.Pos(), "runtime value type switch")
			case *ast.GoStmt:
				add(file, n.Pos(), "runtime goroutine launch")
			case *ast.ChanType:
				add(file, n.Pos(), "runtime channel type")
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "any" {
					add(file, n.Pos(), "conversion to any")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, iterQualifiers, "Pull") {
					add(file, n.Pos(), "iter.Pull in runtime source")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, fmtQualifiers, "Print") {
					add(file, n.Pos(), "implicit I/O call")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, fmtQualifiers, "Printf") {
					add(file, n.Pos(), "implicit I/O call")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, fmtQualifiers, "Println") {
					add(file, n.Pos(), "implicit I/O call")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, fmtQualifiers, "Fprint") {
					add(file, n.Pos(), "implicit I/O call")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, fmtQualifiers, "Fprintf") {
					add(file, n.Pos(), "implicit I/O call")
				}
				if selector, ok := n.Fun.(*ast.SelectorExpr); ok && isQualifiedSelector(selector, fmtQualifiers, "Fprintln") {
					add(file, n.Pos(), "implicit I/O call")
				}
			case *ast.SelectorExpr:
				if isQualifiedSelector(n, syncQualifiers, "Pool") {
					add(file, n.Pos(), "sync.Pool in runtime source")
				}
			}
			if call, ok := node.(*ast.CallExpr); ok && len(call.Args) == 1 && info != nil {
				if typed, ok := info.Types[call.Fun]; ok && typed.IsType() {
					if target, ok := types.Unalias(typed.Type).Underlying().(*types.Interface); ok && target.Empty() {
						add(file, call.Pos(), "input conversion to empty interface")
					}
				}
			}
			return true
		})
		for _, declaration := range file.Decls {
			if group, ok := declaration.(*ast.GenDecl); ok {
				if group.Tok == token.VAR {
					add(file, group.Pos(), "package-level mutable state")
				}
				for _, spec := range group.Specs {
					switch item := spec.(type) {
					case *ast.TypeSpec:
						if outOfScopeExport(item.Name.Name) {
							add(file, item.Name.Pos(), "out-of-scope runtime export "+item.Name.Name)
						}
					case *ast.ValueSpec:
						for _, name := range item.Names {
							if outOfScopeExport(name.Name) {
								add(file, name.Pos(), "out-of-scope runtime export "+name.Name)
							}
						}
					}
				}
			}
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if function.Name != nil && outOfScopeExport(function.Name.Name) {
				add(file, function.Name.Pos(), "out-of-scope runtime export "+function.Name.Name)
			}
			functionName := ""
			if function.Name != nil {
				functionName = function.Name.Name
			}
			failureSide := isFailureInspectionFunction(functionName)
			if function.Type != nil {
				inspectSourceNode(file, function.Type, functionName, failureSide, iterQualifiers, info, add)
			}
			if function.Body != nil {
				inspectSourceNode(file, function.Body, functionName, failureSide, iterQualifiers, info, add)
				checkCapturedMutation(file, function, add, info)
			}
			if directMultiUnwrap(function) {
				add(file, function.Pos(), "multi-error Unwrap returns caller-owned slice")
			}
			if unfilteredAggregateConstruction(function) {
				add(file, function.Pos(), "aggregate stores unfiltered child slice")
			}
			if unguardedErrorAppend(function) {
				add(file, function.Pos(), "error aggregate appends nil child")
			}
			if multiUnwrapReturnsReceiver(function) {
				add(file, function.Pos(), "multi-error Unwrap returns receiver")
			}
		}
	}
	return violations
}

func inspectSourceNode(file *ast.File, root ast.Node, functionName string, failureSide bool, iterQualifiers map[string]struct{}, info *types.Info, add func(*ast.File, token.Pos, string)) {
	ast.Inspect(root, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok && !failureSide {
				if id.Name == "walkIssues" {
					add(file, n.Pos(), "reporting iterator called from validation function "+functionName)
				} else if id.Name == "Issues" || id.Name == "Format" || id.Name == "FormatPath" {
					add(file, n.Pos(), "reporting function called from validation function "+functionName)
				}
			}
		case *ast.TypeAssertExpr:
			if !failureSide {
				if isUnwrapInterface(n.Type) {
					add(file, n.Pos(), "error Unwrap assertion outside failure-side inspection")
				} else {
					add(file, n.Pos(), "value type assertion")
				}
			}
		case *ast.IndexExpr:
			if isQualifiedSelector(n.X, iterQualifiers, "Seq") && !failureSide {
				add(file, n.Pos(), "iter.Seq outside failure-side inspection")
			}
		case *ast.IndexListExpr:
			if isQualifiedSelector(n.X, iterQualifiers, "Seq") && !failureSide {
				add(file, n.Pos(), "iter.Seq outside failure-side inspection")
			}
		case *ast.FuncType:
			if !failureSide && hasReporterParameter(n, info) {
				add(file, n.Pos(), "reporter callback in validation ABI")
			}
		}
		return true
	})
}

func importQualifiers(file *ast.File, path string) map[string]struct{} {
	qualifiers := make(map[string]struct{})
	for _, spec := range file.Imports {
		if strings.Trim(spec.Path.Value, `"`) != path {
			continue
		}
		if spec.Name != nil {
			if spec.Name.Name != "_" && spec.Name.Name != "." {
				qualifiers[spec.Name.Name] = struct{}{}
			}
			continue
		}
		qualifiers[filepath.Base(path)] = struct{}{}
	}
	return qualifiers
}

func isQualifiedSelector(expression ast.Expr, qualifiers map[string]struct{}, name string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector == nil || selector.Sel == nil || selector.Sel.Name != name {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = qualifiers[ident.Name]
	return ok
}

func isFailureInspectionFunction(name string) bool {
	switch name {
	case "Issues", "Format", "FormatPath", "walkIssues", "walkIssueNode", "Error":
		return true
	default:
		return false
	}
}

func isUnwrapInterface(expr ast.Expr) bool {
	interfaceType, ok := expr.(*ast.InterfaceType)
	if !ok || interfaceType.Methods == nil {
		return false
	}
	for _, field := range interfaceType.Methods.List {
		for _, name := range field.Names {
			if name.Name == "Unwrap" {
				return true
			}
		}
	}
	return false
}

func hasReporterParameter(functionType *ast.FuncType, info *types.Info) bool {
	if functionType.Params == nil {
		return false
	}
	for _, field := range functionType.Params.List {
		for _, name := range field.Names {
			if name.Name == "report" || name.Name == "reporter" {
				return true
			}
		}
		if isReporterType(field.Type, info) {
			return true
		}
	}
	return false
}

func isReporterType(expression ast.Expr, info *types.Info) bool {
	if callback, ok := expression.(*ast.FuncType); ok && isErrorToBoolCallback(callback) {
		return true
	}
	if info == nil {
		return false
	}
	var value types.Type
	if typed, ok := info.Types[expression]; ok {
		value = typed.Type
	} else if ident, ok := expression.(*ast.Ident); ok && info.Uses[ident] != nil {
		value = info.Uses[ident].Type()
	}
	if value == nil {
		return false
	}
	signature, ok := types.Unalias(value).Underlying().(*types.Signature)
	return ok && signature.Params().Len() == 1 && signature.Results().Len() == 1 &&
		types.Identical(signature.Params().At(0).Type(), types.Universe.Lookup("error").Type()) &&
		types.Identical(signature.Results().At(0).Type(), types.Typ[types.Bool])
}

func isErrorToBoolCallback(functionType *ast.FuncType) bool {
	if functionType == nil || functionType.Params == nil || functionType.Results == nil {
		return false
	}
	if fieldCount(functionType.Params.List) != 1 || fieldCount(functionType.Results.List) != 1 {
		return false
	}
	parameter := functionType.Params.List[0].Type
	result := functionType.Results.List[0].Type
	return namedType(parameter, "error") && namedType(result, "bool")
}

func fieldCount(fields []*ast.Field) int {
	count := 0
	for _, field := range fields {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func namedType(expression ast.Expr, name string) bool {
	ident, ok := expression.(*ast.Ident)
	return ok && ident.Name == name
}

func directMultiUnwrap(function *ast.FuncDecl) bool {
	if function == nil || function.Name == nil || function.Name.Name != "Unwrap" || function.Type.Results == nil || function.Body == nil {
		return false
	}
	returnsErrorSlice := false
	for _, field := range function.Type.Results.List {
		if errorSliceType(field.Type) {
			returnsErrorSlice = true
		}
	}
	if !returnsErrorSlice {
		return false
	}
	direct := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		result, ok := node.(*ast.ReturnStmt)
		if !ok || len(result.Results) != 1 {
			return true
		}
		switch value := result.Results[0].(type) {
		case *ast.Ident:
			direct = value.Name != "nil"
		case *ast.SelectorExpr, *ast.CompositeLit:
			direct = true
		}
		return true
	})
	return direct
}

func unfilteredAggregateConstruction(function *ast.FuncDecl) bool {
	if function == nil || function.Name == nil || function.Name.Name != "combine" || function.Body == nil {
		return false
	}
	bad := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if bad {
			return false
		}
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range literal.Elts {
			keyed, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := keyed.Key.(*ast.Ident)
			if !ok || key.Name != "children" {
				continue
			}
			value, ok := keyed.Value.(*ast.Ident)
			if ok && value.Name == "children" {
				bad = true
				return false
			}
		}
		return true
	})
	return bad
}

func unguardedErrorAppend(function *ast.FuncDecl) bool {
	if function == nil || function.Name == nil || function.Name.Name != "appendFailure" || function.Body == nil {
		return false
	}
	var visit func(ast.Node, bool)
	bad := false
	visit = func(node ast.Node, guarded bool) {
		if node == nil || bad {
			return
		}
		switch current := node.(type) {
		case *ast.IfStmt:
			guardedBody := guarded || errorNonNilCondition(current.Cond)
			visit(current.Body, guardedBody)
			if current.Else != nil {
				visit(current.Else, guarded || errorNilCondition(current.Cond))
			}
			return
		case *ast.CallExpr:
			if !guarded && len(current.Args) >= 2 && isIdent(current.Fun, "append") && isIdent(current.Args[0], "children") && isIdent(current.Args[1], "err") {
				bad = true
				return
			}
		}
		ast.Inspect(node, func(child ast.Node) bool {
			if child == node {
				return true
			}
			visit(child, guarded)
			return false
		})
	}
	visit(function.Body, false)
	return bad
}

func errorNonNilCondition(expression ast.Expr) bool {
	binary, ok := expression.(*ast.BinaryExpr)
	return ok && binary.Op == token.NEQ && (isIdent(binary.X, "err") && isNilIdent(binary.Y) || isIdent(binary.Y, "err") && isNilIdent(binary.X))
}

func errorNilCondition(expression ast.Expr) bool {
	binary, ok := expression.(*ast.BinaryExpr)
	return ok && binary.Op == token.EQL && (isIdent(binary.X, "err") && isNilIdent(binary.Y) || isIdent(binary.Y, "err") && isNilIdent(binary.X))
}

func isNilIdent(expression ast.Expr) bool { return isIdent(expression, "nil") }

func isIdent(expression ast.Expr, name string) bool {
	ident, ok := expression.(*ast.Ident)
	return ok && ident.Name == name
}

func multiUnwrapReturnsReceiver(function *ast.FuncDecl) bool {
	if function == nil || function.Name == nil || function.Name.Name != "Unwrap" || function.Body == nil || function.Recv == nil {
		return false
	}
	receiverName := ""
	for _, field := range function.Recv.List {
		if len(field.Names) > 0 {
			receiverName = field.Names[0].Name
			break
		}
	}
	if receiverName == "" {
		return false
	}
	bad := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if bad {
			return false
		}
		switch current := node.(type) {
		case *ast.ReturnStmt:
			for _, result := range current.Results {
				if isIdent(result, receiverName) {
					bad = true
					return false
				}
			}
		case *ast.CallExpr:
			for _, argument := range current.Args {
				if isIdent(argument, receiverName) {
					bad = true
					return false
				}
			}
		case *ast.CompositeLit:
			for _, element := range current.Elts {
				if keyed, ok := element.(*ast.KeyValueExpr); ok {
					if isIdent(keyed.Value, receiverName) {
						bad = true
						return false
					}
					continue
				}
				if isIdent(element, receiverName) {
					bad = true
					return false
				}
			}
		}
		return true
	})
	return bad
}

func errorSliceType(expr ast.Expr) bool {
	array, ok := expr.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	ident, ok := array.Elt.(*ast.Ident)
	return ok && ident.Name == "error"
}

func checkCapturedMutation(file *ast.File, function *ast.FuncDecl, add func(*ast.File, token.Pos, string), info *types.Info) {
	if info == nil || function.Body == nil {
		return
	}
	literals := make(map[types.Object]*ast.FuncLit)
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch declaration := node.(type) {
		case *ast.AssignStmt:
			for index, left := range declaration.Lhs {
				if index >= len(declaration.Rhs) {
					break
				}
				literal, ok := declaration.Rhs[index].(*ast.FuncLit)
				if !ok {
					continue
				}
				if object := objectForExpression(left, info); object != nil {
					literals[object] = literal
				}
			}
		case *ast.ValueSpec:
			for index, name := range declaration.Names {
				if index >= len(declaration.Values) {
					break
				}
				literal, ok := declaration.Values[index].(*ast.FuncLit)
				if !ok {
					continue
				}
				if object := objectForExpression(name, info); object != nil {
					literals[object] = literal
				}
			}
		}
		return true
	})
	seen := make(map[*ast.FuncLit]struct{})
	inspect := func(literal *ast.FuncLit) {
		if literal == nil {
			return
		}
		if _, ok := seen[literal]; ok {
			return
		}
		seen[literal] = struct{}{}
		inspectCapturedMutation(file, literal, add, info)
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		returnNode, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, result := range returnNode.Results {
			if literal, ok := result.(*ast.FuncLit); ok {
				inspect(literal)
				continue
			}
			if object := objectForExpression(result, info); object != nil {
				inspect(literals[object])
			}
		}
		return true
	})
}

func objectForExpression(expression ast.Expr, info *types.Info) types.Object {
	ident, ok := expression.(*ast.Ident)
	if !ok || info == nil {
		return nil
	}
	if object := info.Uses[ident]; object != nil {
		return object
	}
	return info.Defs[ident]
}

func inspectCapturedMutation(file *ast.File, literal *ast.FuncLit, add func(*ast.File, token.Pos, string), info *types.Info) {
	mutated := false
	mark := func(node ast.Node) {
		if !mutated {
			add(file, node.Pos(), "mutable state captured by returned rule")
			mutated = true
		}
	}
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		if mutated {
			return false
		}
		switch n := node.(type) {
		case *ast.AssignStmt:
			for _, left := range n.Lhs {
				if capturedRoot(left, literal, info) {
					mark(n)
					break
				}
			}
		case *ast.IncDecStmt:
			if capturedRoot(n.X, literal, info) {
				mark(n)
			}
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "make" && len(n.Args) > 0 {
				if _, ok := n.Args[0].(*ast.MapType); ok {
					add(file, n.Pos(), "per-call map inside returned rule")
				}
			}
			if id, ok := n.Fun.(*ast.Ident); ok && (id.Name == "append" || id.Name == "copy" || id.Name == "clear" || id.Name == "delete") && len(n.Args) > 0 && capturedRoot(n.Args[0], literal, info) {
				mark(n)
			}
			if selector, ok := n.Fun.(*ast.SelectorExpr); ok && mutableMethod(selector.Sel.Name) && capturedRoot(selector.X, literal, info) {
				mark(n)
			}
		}
		return true
	})
}

func mutableMethod(name string) bool {
	switch name {
	case "Store", "LoadOrStore", "Delete", "Swap", "CompareAndSwap", "Add", "Append", "Write", "WriteString":
		return true
	default:
		return false
	}
}

func capturedRoot(expr ast.Expr, literal *ast.FuncLit, info *types.Info) bool {
	for {
		switch value := expr.(type) {
		case *ast.Ident:
			object := info.Uses[value]
			if object == nil {
				object = info.Defs[value]
			}
			return objectOutsideLiteral(object, literal)
		case *ast.IndexExpr:
			expr = value.X
		case *ast.IndexListExpr:
			expr = value.X
		case *ast.SelectorExpr:
			expr = value.X
		case *ast.StarExpr:
			expr = value.X
		default:
			return false
		}
	}
}

func objectOutsideLiteral(object types.Object, literal *ast.FuncLit) bool {
	if object == nil || literal == nil {
		return false
	}
	if object.Pos() < literal.Pos() {
		return true
	}
	variable, ok := object.(*types.Var)
	if !ok || variable.Parent() == nil {
		return false
	}
	// A package variable can be declared after the constructor. Its lexical
	// position is therefore not enough to identify it as captured state.
	return variable.Parent().Parent() == nil
}

func strconvQuote(value string) string {
	return `"` + value + `"`
}

func isAllowedRuntimeImport(path string) bool {
	switch path {
	case "bytes", "iter", "math", "slices", "strconv", "strings", "time", "unicode/utf8":
		return true
	}
	return false
}

func outOfScopeExport(name string) bool {
	if name == "Format" || name == "FormatPath" {
		return false
	}
	if name == "Fatal" || name == "RuleStopOnError" || name == "RuleNot" || name == "Or" {
		return true
	}
	return strings.Contains(name, "Context") || strings.Contains(name, "Regex") ||
		strings.Contains(name, "Regexp") || strings.Contains(name, "Tag") ||
		strings.HasPrefix(name, "Format")
}
