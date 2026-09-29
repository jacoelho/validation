package validation

import (
	"errors"
	"testing"
)

type countedUnwrap struct {
	calls *int
	child error
}

func (e countedUnwrap) Error() string { return "counted" }
func (e countedUnwrap) Unwrap() error { *e.calls++; return e.child }

func TestWalkIssuesEarlyStopAndReplay(t *testing.T) {
	calls := 0
	root := combine([]error{NewViolation("first", nil), countedUnwrap{&calls, errors.New("last")}})
	seen := 0
	for range walkIssues(root) {
		seen++
		break
	}
	if seen != 1 || calls != 0 {
		t.Fatalf("early stop seen=%d later unwrap calls=%d", seen, calls)
	}
	for range walkIssues(root) {
		seen++
	}
	if seen != 3 || calls != 1 {
		t.Fatalf("replay seen=%d later unwrap calls=%d", seen, calls)
	}
}

func TestWalkIssuesMixedTreeAndNil(t *testing.T) {
	cause := errors.New("coded cause")
	first := errors.New("first external")
	second := errors.New("second external")
	coded := NewViolation("coded", cause)
	root := at(Segment{Kind: FieldSegment, Name: "outer"},
		combine([]error{coded, errors.Join(first, second)}))
	var got []Issue
	for issue := range walkIssues(root) {
		got = append(got, issue)
	}
	if len(got) != 3 {
		t.Fatalf("walk yielded %d issues, want 3", len(got))
	}
	wantCodes := []Code{"coded", CodeExternal, CodeExternal}
	wantErrors := []error{coded, first, second}
	for i, issue := range got {
		if issue.Code != wantCodes[i] || issue.Err != wantErrors[i] || FormatPath(issue.Path) != "$.outer" {
			t.Fatalf("issue %d = %+v, want code %q, error %v at $.outer", i, issue, wantCodes[i], wantErrors[i])
		}
	}
	for range walkIssues(nil) {
		t.Fatal("nil error yielded an issue")
	}
}

type panicUnwrap struct{}

func (panicUnwrap) Error() string { return "later" }
func (panicUnwrap) Unwrap() error { panic("later sibling inspected") }

func TestWalkIssuesStopBeforePanickingSibling(t *testing.T) {
	root := combine([]error{NewViolation("first", nil), panicUnwrap{}})
	seen := 0
	for range walkIssues(root) {
		seen++
		break
	}
	if seen != 1 {
		t.Fatalf("yielded %d issues, want 1", seen)
	}
}

func TestWalkIssuesRetainedPathsAndRepeatedIdentity(t *testing.T) {
	shared := errors.New("same sentinel")
	root := combine([]error{
		at(Segment{Kind: FieldSegment, Name: "left"}, shared),
		at(Segment{Kind: FieldSegment, Name: "right"}, shared),
		at(Segment{Kind: FieldSegment, Name: "third"}, shared),
	})
	walk := walkIssues(root)
	var first []Issue
	for issue := range walk {
		first = append(first, issue)
		break
	}
	if len(first) != 1 || first[0].Err != shared {
		t.Fatalf("first traversal=%+v", first)
	}
	var all []Issue
	for issue := range walk {
		all = append(all, issue)
	}
	if len(all) != 3 {
		t.Fatalf("replay yielded %d, want 3", len(all))
	}
	if FormatPath(all[0].Path) != "$.left" || FormatPath(all[1].Path) != "$.right" || FormatPath(all[2].Path) != "$.third" {
		t.Fatalf("paths=%+v", all)
	}
	all[0].Path[0].Name = "changed"
	if FormatPath(all[1].Path) != "$.right" || FormatPath(Issues(root)[0].Path) != "$.left" {
		t.Fatal("yielded paths aliased")
	}
}

func TestAT_ERRORS_009_ErrorTreeContract(t *testing.T) {
	shared := errors.New("same sentinel")
	left := at(Segment{Kind: FieldSegment, Name: "left"}, shared)
	right := at(Segment{Kind: FieldSegment, Name: "right"}, shared)
	children := []error{nil, left, nil, right, nil}
	root := combine(children)
	group, ok := root.(*aggregate)
	if !ok {
		t.Fatalf("combine returned %T, want aggregate", root)
	}
	children[1], children[3] = nil, nil
	unwrapped := group.Unwrap()
	if len(unwrapped) != 2 || unwrapped[0] != left || unwrapped[1] != right {
		t.Fatalf("aggregate children = %v, want the two original non-nil errors", unwrapped)
	}
	unwrapped[0] = nil
	if group.Unwrap()[0] != left {
		t.Fatal("Unwrap exposed mutable aggregate storage")
	}
	issues := Issues(root)
	if len(issues) != 2 || issues[0].Err != shared || issues[1].Err != shared ||
		FormatPath(issues[0].Path) != "$.left" || FormatPath(issues[1].Path) != "$.right" {
		t.Fatalf("repeated sentinel occurrences = %+v", issues)
	}
	if combine([]error{nil, nil}) != nil || combine([]error{nil, left, nil}) != left || at(Segment{Kind: FieldSegment, Name: "ignored"}, nil) != nil {
		t.Fatal("library construction retained a nil child")
	}
}
