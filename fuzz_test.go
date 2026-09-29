package validation_test

import (
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	v "github.com/jacoelho/validation/v2"
)

// decodePath is a test-only inverse of the documented grammar. It does not use
// the validation package's formatting or traversal helpers.
func decodePath(text string) ([]v.Segment, bool) {
	if !strings.HasPrefix(text, "$") {
		return nil, false
	}
	var segments []v.Segment
	for i := 1; i < len(text); {
		if text[i] == '.' {
			i++
			if i < len(text) && text[i] == '[' {
				name, next, ok := decodeQuotedSegment(text, i)
				if !ok {
					return nil, false
				}
				segments = append(segments, v.Segment{Kind: v.FieldSegment, Name: name})
				i = next
				continue
			}
			start := i
			for i < len(text) && (text[i] == '_' || text[i] >= 'A' && text[i] <= 'Z' || text[i] >= 'a' && text[i] <= 'z' || i > start && text[i] >= '0' && text[i] <= '9') {
				i++
			}
			if i == start {
				return nil, false
			}
			segments = append(segments, v.Segment{Kind: v.FieldSegment, Name: text[start:i]})
			continue
		}
		if text[i] != '[' {
			return nil, false
		}
		if i+1 < len(text) && text[i+1] == '"' {
			name, next, ok := decodeQuotedSegment(text, i)
			if !ok {
				return nil, false
			}
			segments = append(segments, v.Segment{Kind: v.KeySegment, Name: name})
			i = next
			continue
		}
		end := strings.IndexByte(text[i:], ']')
		if end < 0 {
			return nil, false
		}
		end += i
		n, err := strconv.Atoi(text[i+1 : end])
		if err != nil || n < 0 {
			return nil, false
		}
		segments = append(segments, v.Segment{Kind: v.IndexSegment, Index: n})
		i = end + 1
	}
	return segments, true
}
func decodeQuotedSegment(text string, opening int) (string, int, bool) {
	if text[opening] != '[' {
		return "", 0, false
	}
	for end := opening + 2; end < len(text); end++ {
		if text[end] != ']' {
			continue
		}
		name, err := strconv.Unquote(text[opening+1 : end])
		if err == nil {
			return name, end + 1, true
		}
	}
	return "", 0, false
}

func FuzzPaths(f *testing.F) {
	for _, seed := range [][]byte{nil, []byte("a.b"), []byte{0xff, 0, 1, 2}, []byte("name\n\"[]")} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64 {
			data = data[:64]
		}
		var path []v.Segment
		for i := 0; i < len(data) && len(path) < 16; {
			kind := data[i] % 3
			i++
			if kind == 1 {
				path = append(path, v.Segment{Kind: v.IndexSegment, Index: int(data[i%len(data)])})
				continue
			}
			length := 0
			if i < len(data) {
				length = int(data[i] % 5)
				i++
			}
			if length > len(data)-i {
				length = len(data) - i
			}
			name := string(data[i : i+length])
			i += length
			if kind == 0 && name == "" {
				name = "_"
			}
			segment := v.Segment{Kind: v.FieldSegment, Name: name}
			if kind == 2 {
				segment.Kind = v.KeySegment
			}
			path = append(path, segment)
		}
		before := append([]v.Segment(nil), path...)
		got, ok := decodePath(v.FormatPath(path))
		if !ok || !reflect.DeepEqual(got, path) && !(len(got) == 0 && len(path) == 0) {
			t.Fatalf("round trip path %#v -> %q -> %#v (ok=%t)", path, v.FormatPath(path), got, ok)
		}
		if !reflect.DeepEqual(path, before) && !(len(path) == 0 && len(before) == 0) {
			t.Fatalf("formatting changed path: before=%#v after=%#v", before, path)
		}
	})
}

func FuzzSliceOccurrences(f *testing.F) {
	for _, seed := range [][]byte{nil, {0}, {0, 0}, {1, 2, 1, 2, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64 {
			data = data[:64]
		}
		values := append([]byte(nil), data...)
		rule := v.All(v.SliceOneOf[[]byte](1, 2), v.SliceUnique[[]byte]())
		issues := v.Issues(rule(values))
		var wantCodes []v.Code
		var wantIndices []int
		for i, x := range data {
			if x != 1 && x != 2 {
				wantCodes = append(wantCodes, v.CodeOneOf)
				wantIndices = append(wantIndices, i)
			}
		}
		for i, x := range data {
			for j := 0; j < i; j++ {
				if data[j] == x {
					wantCodes = append(wantCodes, v.CodeUnique)
					wantIndices = append(wantIndices, i)
					break
				}
			}
		}
		if len(issues) != len(wantCodes) {
			t.Fatalf("issues=%d, want %d for %v", len(issues), len(wantCodes), data)
		}
		for i, issue := range issues {
			if issue.Code != wantCodes[i] || len(issue.Path) != 1 || issue.Path[0].Kind != v.IndexSegment || issue.Path[0].Index != wantIndices[i] {
				t.Fatalf("issue[%d]=%+v, want %s at %d", i, issue, wantCodes[i], wantIndices[i])
			}
		}
		if !reflect.DeepEqual(values, append([]byte(nil), data...)) {
			t.Fatal("input changed")
		}
	})
}

func FuzzMapIssueOrder(f *testing.F) {
	for _, seed := range [][]byte{nil, {1, 0, 2, 7}, {15, 255, 0, 6, 1, 8}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64 {
			data = data[:64]
		}
		var present [16]bool
		var values [16]int
		for i := 0; i+1 < len(data); i += 2 {
			key := int(data[i] % 16)
			present[key] = true
			values[key] = int(int8(data[i+1]))
		}
		forward := make(map[string]int)
		reverse := make(map[string]int)
		for i := 0; i < 16; i++ {
			if present[i] {
				forward[string(rune('a'+i))] = values[i]
			}
		}
		for i := 15; i >= 0; i-- {
			if present[i] {
				reverse[string(rune('a'+i))] = values[i]
			}
		}
		before := make(map[string]int, len(forward))
		reverseBefore := make(map[string]int, len(reverse))
		for key, value := range forward {
			before[key] = value
		}
		for key, value := range reverse {
			reverseBefore[key] = value
		}
		rule := v.MapValues[map[string]int](v.StringKeys[string](), v.Min(0), v.Max(5))
		var wantCodes []v.Code
		var wantKeys []string
		for i := 0; i < 16; i++ {
			if !present[i] {
				continue
			}
			key := string(rune('a' + i))
			if values[i] < 0 {
				wantCodes = append(wantCodes, v.CodeMin)
				wantKeys = append(wantKeys, key)
			}
			if values[i] > 5 {
				wantCodes = append(wantCodes, v.CodeMax)
				wantKeys = append(wantKeys, key)
			}
		}
		for _, input := range []map[string]int{forward, reverse} {
			issues := v.Issues(rule(input))
			if len(issues) != len(wantCodes) {
				t.Fatalf("issue count=%d, want %d for %v", len(issues), len(wantCodes), input)
			}
			for i, issue := range issues {
				if issue.Code != wantCodes[i] || len(issue.Path) != 1 || issue.Path[0].Kind != v.KeySegment || issue.Path[0].Name != wantKeys[i] {
					t.Fatalf("issue[%d]=%+v, want %s at %q", i, issue, wantCodes[i], wantKeys[i])
				}
			}
		}
		if !reflect.DeepEqual(before, forward) || !reflect.DeepEqual(reverseBefore, reverse) {
			t.Fatalf("map input changed: forward=%v reverse=%v", forward, reverse)
		}
	})
}

func FuzzNumericBounds(f *testing.F) {
	f.Add(int64(0), int64(0), int64(0))
	f.Add(int64(-9223372036854775808), int64(9223372036854775807), int64(0))
	f.Add(int64(9223372036854775807), int64(-9223372036854775808), int64(1))
	f.Fuzz(func(t *testing.T, lower, upper, value int64) {
		lo := big.NewInt(lower)
		hi := big.NewInt(upper)
		if lo.Cmp(hi) > 0 {
			lo, hi = hi, lo
			lower, upper = upper, lower
		}
		rule := v.All(v.Min(lower), v.Max(upper), v.Between(lower, upper))
		issues := v.Issues(rule(value))
		x := big.NewInt(value)
		var want []v.Code
		if x.Cmp(lo) < 0 {
			want = append(want, v.CodeMin)
		}
		if x.Cmp(hi) > 0 {
			want = append(want, v.CodeMax)
		}
		if x.Cmp(lo) < 0 || x.Cmp(hi) > 0 {
			want = append(want, v.CodeBetween)
		}
		if len(issues) != len(want) {
			t.Fatalf("%d outside [%d,%d]: issues=%v, want %v", value, lower, upper, issues, want)
		}
		for i, issue := range issues {
			if issue.Code != want[i] || len(issue.Path) != 0 {
				t.Fatalf("issue[%d]=%+v, want root %s", i, issue, want[i])
			}
		}
	})
}
