// allocprobe measures successful validation in a fresh process.
//
// The checker builds this program as an optimised binary and runs one fixture
// per process. Rule construction and fixture preparation happen before the
// MemStats snapshots; only calls to the stored Rule are measured.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
	"unicode/utf8"

	v "github.com/jacoelho/validation/v2"
)

type fixtureKind string

const (
	allocationFree fixtureKind = "allocation-free"
	control        fixtureKind = "control"
)

type fixtureInfo struct {
	Name string      `json:"name"`
	Kind fixtureKind `json:"kind"`
}

type preparedFixture struct {
	fixtureInfo
	run func() error
}

type namedUint uint64
type namedFloat float64
type namedBool bool
type namedString string
type namedBytes []byte
type namedInts []int
type namedRecords []record
type namedMap map[string]string
type namedIntMap map[int]namedString
type namedArray [8]int

type record struct {
	Value int
}

type flat16 struct {
	F0, F1, F2, F3     int
	F4, F5, F6, F7     int
	F8, F9, F10, F11   int
	F12, F13, F14, F15 int
}

type nestedNode struct {
	Value int
	Next  *nestedNode
}

type arrayParent struct {
	Values namedArray
}

type integerExtremes struct {
	SignedMin   int
	SignedMax   int
	UnsignedMin uint
	UnsignedMax uint
}

type membershipInput struct {
	Scalar int
	Values []int
}

type conditionalInput struct {
	Enabled bool
	Value   int
}

type comparableInput struct {
	Flag  namedBool
	Label namedString
}

type report struct {
	Fixture string      `json:"fixture"`
	Kind    fixtureKind `json:"kind"`
	Phase   string      `json:"phase"`
	Calls   int         `json:"calls"`
	Mallocs uint64      `json:"mallocs"`
	Bytes   uint64      `json:"bytes"`
	Go      string      `json:"go"`
	OS      string      `json:"os"`
	Arch    string      `json:"arch"`
}

// retained makes deliberate control allocations observable and keeps them
// live while the process is measured. It is never touched by library
// fixtures.
var retained []byte

func stored[T any](rule v.Rule[T], input T) func() error {
	return func() error { return rule(input) }
}

func all[T any](rules ...v.Rule[T]) v.Rule[T] { return v.All(rules...) }

func scalarSigned() func() error {
	input := 2
	rule := all(
		v.Min[int](1),
		v.Max[int](3),
		v.Between[int](1, 3),
		v.GreaterThan[int](1),
		v.LessThan[int](3),
		v.Positive[int](),
		v.NonNegative[int](),
		v.NotZero[int](),
		v.Equal[int](2),
		v.NotEqual[int](0),
		v.OneOf[int](1, 2, 3),
		v.NotOneOf[int](0, 4),
	)
	return stored(rule, input)
}

func genericMethods() func() error {
	input := conditionalInput{Enabled: true, Value: 2}
	get := func(x conditionalInput) int { return x.Value }
	getPresent := func(x conditionalInput) (int, bool) { return x.Value, x.Enabled }
	child := v.Min(0)
	rule := v.All(
		child.Field("value", get),
		child.Project(get),
		child.OptionalValue(getPresent),
		child.RequiredValue(getPresent),
	)
	return stored(rule, input)
}

func scalarExtremes() func() error {
	signedMin := -int(^uint(0)>>1) - 1
	signedMax := int(^uint(0) >> 1)
	unsignedMax := ^uint(0)
	input := integerExtremes{
		SignedMin:   signedMin,
		SignedMax:   signedMax,
		UnsignedMin: 0,
		UnsignedMax: unsignedMax,
	}
	rule := all(
		v.Field("signed_min", func(x integerExtremes) int { return x.SignedMin }, v.Min[int](signedMin), v.Negative[int]()),
		v.Field("signed_max", func(x integerExtremes) int { return x.SignedMax }, v.Max[int](signedMax), v.Positive[int]()),
		v.Field("unsigned_min", func(x integerExtremes) uint { return x.UnsignedMin }, v.Zero[uint](), v.NonNegative[uint]()),
		v.Field("unsigned_max", func(x integerExtremes) uint { return x.UnsignedMax }, v.Max[uint](unsignedMax), v.Positive[uint]()),
	)
	return stored(rule, input)
}

func scalarZero() func() error {
	rule := all(
		v.Zero[int](),
		v.NonNegative[int](),
		v.NonPositive[int](),
		v.Equal[int](0),
		v.NotEqual[int](1),
		v.OneOf[int](0, 1),
		v.NotOneOf[int](1, 2),
	)
	return stored(rule, 0)
}

func scalarNegative() func() error {
	input := -2
	rule := all(
		v.Min[int](-3),
		v.Max[int](-1),
		v.Between[int](-3, -1),
		v.GreaterThan[int](-3),
		v.LessThan[int](-1),
		v.Negative[int](),
		v.NonPositive[int](),
		v.NotZero[int](),
	)
	return stored(rule, input)
}

func scalarUnsigned() func() error {
	input := namedUint(7)
	rule := all(
		v.Min[namedUint](1),
		v.Max[namedUint](9),
		v.Between[namedUint](1, 9),
		v.GreaterThan[namedUint](1),
		v.LessThan[namedUint](9),
		v.Positive[namedUint](),
		v.NonNegative[namedUint](),
		v.NotZero[namedUint](),
	)
	return stored(rule, input)
}

func scalarFloat() func() error {
	input := namedFloat(1.5)
	rule := all(
		v.Min[namedFloat](1),
		v.Max[namedFloat](2),
		v.Between[namedFloat](1, 2),
		v.GreaterThan[namedFloat](1),
		v.LessThan[namedFloat](2),
		v.Positive[namedFloat](),
		v.NonNegative[namedFloat](),
		v.NotZero[namedFloat](),
		v.NotNaN[namedFloat](),
		v.Finite[namedFloat](),
	)
	return stored(rule, input)
}

func scalarInfinity() func() error {
	return stored(v.All(v.NotNaN[float64](), v.Positive[float64]()), math.Inf(1))
}

func scalarSignedZero() func() error {
	input := math.Copysign(0, -1)
	rule := all(v.Zero[float64](), v.NonNegative[float64](), v.NonPositive[float64](), v.NotNaN[float64](), v.Finite[float64]())
	return stored(rule, input)
}

func scalarComparable() func() error {
	input := comparableInput{Flag: true, Label: "ready"}
	rule := v.Struct(
		v.Field("flag", func(x comparableInput) namedBool { return x.Flag },
			v.Equal[namedBool](true),
			v.NotEqual[namedBool](false),
			v.OneOf[namedBool](true, false),
			v.NotOneOf[namedBool](false),
		),
		v.Field("label", func(x comparableInput) namedString { return x.Label },
			v.Equal[namedString]("ready"),
			v.NotEqual[namedString]("missing"),
			v.OneOf[namedString]("ready", "other"),
			v.NotOneOf[namedString]("missing"),
		),
	)
	return stored(rule, input)
}

func comparator() func() error {
	type score struct{ value int }
	compare := func(left, right score) int {
		switch {
		case left.value < right.value:
			return -1
		case left.value > right.value:
			return 1
		default:
			return 0
		}
	}
	input := score{value: 10}
	rule := all(
		v.MinBy(score{value: 1}, compare),
		v.MaxBy(score{value: 20}, compare),
		v.BetweenBy(score{value: 1}, score{value: 20}, compare),
	)
	return stored(rule, input)
}

func stringCatalogue() func() error {
	input := namedString("prefix-λtarget-suffix")
	byteCount := len(input)
	runeCount := utf8.RuneCountInString(string(input))
	rule := all(
		v.NotEmpty[namedString](),
		v.ByteLength[namedString](byteCount),
		v.ByteMinLength[namedString](byteCount-1),
		v.ByteMaxLength[namedString](byteCount+1),
		v.ByteLengthBetween[namedString](byteCount-1, byteCount+1),
		v.RuneLength[namedString](runeCount),
		v.RuneMinLength[namedString](runeCount-1),
		v.RuneMaxLength[namedString](runeCount+1),
		v.RuneLengthBetween[namedString](runeCount-1, runeCount+1),
		v.Contains[namedString]("target"),
		v.HasPrefix[namedString]("prefix-"),
		v.HasSuffix[namedString]("-suffix"),
		v.UTF8[namedString](),
	)
	return stored(rule, input)
}

func bytesCatalogue() func() error {
	input := namedBytes("prefix-λtarget-suffix")
	byteCount := len(input)
	rule := all(
		v.BytesNotEmpty[namedBytes](),
		v.BytesLength[namedBytes](byteCount),
		v.BytesMinLength[namedBytes](byteCount-1),
		v.BytesMaxLength[namedBytes](byteCount+1),
		v.BytesLengthBetween[namedBytes](byteCount-1, byteCount+1),
		v.BytesContains[namedBytes](namedBytes("target")),
		v.BytesHasPrefix[namedBytes](namedBytes("prefix-")),
		v.BytesHasSuffix[namedBytes](namedBytes("-suffix")),
		v.BytesUTF8[namedBytes](),
		v.BytesEqual[namedBytes](namedBytes("prefix-λtarget-suffix")),
	)
	return stored(rule, input)
}

func timeCatalogue() func() error {
	before := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	input := before.Add(24 * time.Hour)
	after := input.Add(24 * time.Hour)
	rule := all(
		v.TimeBefore(after),
		v.TimeBeforeOrEqual(input),
		v.TimeAfter(before),
		v.TimeAfterOrEqual(input),
		v.TimeBetween(before, after),
		v.TimeNotZero(),
	)
	return stored(rule, input)
}

func flatFields() func() error {
	input := flat16{
		F0: 1, F1: 1, F2: 1, F3: 1, F4: 1, F5: 1, F6: 1, F7: 1,
		F8: 1, F9: 1, F10: 1, F11: 1, F12: 1, F13: 1, F14: 1, F15: 1,
	}
	rule := all(
		v.Field("f0", func(x flat16) int { return x.F0 }, v.Positive[int]()),
		v.Field("f1", func(x flat16) int { return x.F1 }, v.Positive[int]()),
		v.Field("f2", func(x flat16) int { return x.F2 }, v.Positive[int]()),
		v.Field("f3", func(x flat16) int { return x.F3 }, v.Positive[int]()),
		v.Field("f4", func(x flat16) int { return x.F4 }, v.Positive[int]()),
		v.Field("f5", func(x flat16) int { return x.F5 }, v.Positive[int]()),
		v.Field("f6", func(x flat16) int { return x.F6 }, v.Positive[int]()),
		v.Field("f7", func(x flat16) int { return x.F7 }, v.Positive[int]()),
		v.Field("f8", func(x flat16) int { return x.F8 }, v.Positive[int]()),
		v.Field("f9", func(x flat16) int { return x.F9 }, v.Positive[int]()),
		v.Field("f10", func(x flat16) int { return x.F10 }, v.Positive[int]()),
		v.Field("f11", func(x flat16) int { return x.F11 }, v.Positive[int]()),
		v.Field("f12", func(x flat16) int { return x.F12 }, v.Positive[int]()),
		v.Field("f13", func(x flat16) int { return x.F13 }, v.Positive[int]()),
		v.Field("f14", func(x flat16) int { return x.F14 }, v.Positive[int]()),
		v.Field("f15", func(x flat16) int { return x.F15 }, v.Positive[int]()),
	)
	return stored(rule, input)
}

func flatField1() func() error {
	type input struct{ Value int }
	return stored(v.Field("value", func(x input) int { return x.Value }, v.Positive[int]()), input{Value: 1})
}

func structConditions() func() error {
	input := conditionalInput{Enabled: true, Value: 3}
	rule := v.Struct(
		v.Field("enabled", func(x conditionalInput) bool { return x.Enabled }, v.Equal[bool](true)),
		v.When(func(x conditionalInput) bool { return x.Enabled },
			v.Field("value", func(x conditionalInput) int { return x.Value }, v.Positive[int]())),
		v.Unless(func(x conditionalInput) bool { return !x.Enabled },
			v.Field("value_again", func(x conditionalInput) int { return x.Value }, v.Min[int](1))),
	)
	return stored(rule, input)
}

func deepFields(depth int) func() error {
	input := nestedInput(depth)
	rule := v.Field("value", func(x nestedNode) int { return x.Value }, v.Positive[int]())
	for range depth {
		rule = v.Field("next", func(x nestedNode) *nestedNode { return x.Next }, v.RequiredPtr(rule))
	}
	return stored(rule, input)
}

func nestedInput(depth int) nestedNode {
	input := nestedNode{Value: 1}
	for range depth {
		input = nestedNode{Value: 1, Next: &input}
	}
	return input
}

func presenceOptionalAbsent() func() error {
	return stored(v.OptionalPtr(v.Positive[int]()), (*int)(nil))
}

func presenceRequiredPresent() func() error {
	value := 1
	return stored(v.RequiredPtr(v.Positive[int]()), &value)
}

func presenceValueAbsent() func() error {
	return stored(v.OptionalValue(func(int) (int, bool) { return 0, false }, v.Positive[int]()), 0)
}

func presenceValuePresent() func() error {
	return stored(v.RequiredValue(func(int) (int, bool) { return 1, true }, v.Positive[int]()), 0)
}

func presenceValuePresentZero() func() error {
	return stored(v.RequiredValue(func(int) (int, bool) { return 0, true }, v.Zero[int]()), 0)
}

func presenceNested() func() error {
	value := 1
	ptr := &value
	rule := v.OptionalPtr(v.RequiredPtr(v.Positive[int]()))
	return stored(rule, &ptr)
}

func arrayProjection() func() error {
	input := &arrayParent{Values: namedArray{1, 2, 3, 4, 5, 6, 7, 8}}
	rule := v.Project(func(parent *arrayParent) []int { return parent.Values[:] },
		v.SliceLength[[]int](8),
		v.SliceUnique[[]int](),
		v.Each[[]int](v.Positive[int]()),
	)
	return stored(rule, input)
}

func sliceValues(n int) []int {
	values := make([]int, n)
	for i := range values {
		values[i] = i + 1
	}
	return values
}

func sliceBySize(n int) func() error {
	values := namedInts(sliceValues(n))
	rules := []v.Rule[namedInts]{
		v.SliceLength[namedInts](n),
		v.SliceMinLength[namedInts](0),
		v.SliceMaxLength[namedInts](n),
		v.SliceLengthBetween[namedInts](0, n),
		v.Each[namedInts](v.Positive[int]()),
		v.SliceUnique[namedInts](),
		v.SliceNotOneOf[namedInts](0),
		v.SliceOneOf[namedInts](sliceValues(n)...),
	}
	if n > 0 {
		rules = append(rules,
			v.SliceContains[namedInts](1),
			v.AtIndex[namedInts](0, v.Positive[int]()),
		)
	}
	return stored(v.All(rules...), values)
}

func namedNestedSlices() func() error {
	input := []namedInts{{1, 2}, {3, 4, 5}}
	inner := v.All(
		v.SliceMinLength[namedInts](1),
		v.Each[namedInts](v.Positive[int]()),
	)
	rule := v.Each[[]namedInts](inner)
	return stored(rule, input)
}

func sliceOfStructs() func() error {
	input := namedRecords{{Value: 1}, {Value: 2}, {Value: 3}}
	rule := v.Each[namedRecords](v.Field("value", func(item record) int { return item.Value }, v.Positive[int]()))
	return stored(rule, input)
}

func membership(size int) func() error {
	input := membershipInput{}
	if size == 0 {
		input.Scalar = 123
		input.Values = []int{}
		rule := v.Struct(
			v.Field("scalar", func(x membershipInput) int { return x.Scalar }, v.NotOneOf[int]()),
			v.Field("values", func(x membershipInput) []int { return x.Values }, v.SliceOneOf[[]int]()),
		)
		return stored(rule, input)
	}
	allowed := make([]int, size)
	for i := range allowed {
		allowed[i] = i
	}
	input.Scalar = 0
	input.Values = []int{0}
	rule := v.Struct(
		v.Field("scalar", func(x membershipInput) int { return x.Scalar }, v.OneOf[int](allowed...)),
		v.Field("values", func(x membershipInput) []int { return x.Values }, v.SliceOneOf[[]int](allowed...)),
	)
	return stored(rule, input)
}

func stringMapValues(n int) namedMap {
	values := make(namedMap, n)
	for i := 0; i < n; i++ {
		values["k"+strconv.Itoa(i)] = "ok"
	}
	return values
}

func mapRule(n int) func() error {
	input := stringMapValues(n)
	order := v.StringKeys[string]()
	allowed := make([]string, 0, n)
	for i := 0; i < n; i++ {
		allowed = append(allowed, "k"+strconv.Itoa(i))
	}
	entryRule := v.Project(func(entry v.Entry[string, string]) string { return entry.Value }, v.NotEmpty[string]())
	rules := []v.Rule[namedMap]{
		v.MapLength[namedMap](n),
		v.MapMinLength[namedMap](0),
		v.MapMaxLength[namedMap](n),
		v.MapLengthBetween[namedMap](0, n),
		v.MapKeys[namedMap](order, v.NotEmpty[string]()),
		v.MapValues[namedMap](order, v.NotEmpty[string]()),
		v.MapEach[namedMap](order, entryRule),
		v.MapKeysOneOf[namedMap](order, allowed...),
		v.MapKeysNotOneOf[namedMap](order, "missing"),
	}
	if n == 0 {
		rules = append(rules, v.MapOptionalKey[namedMap]("missing", order, v.NotEmpty[string]()))
	} else {
		rules = append(rules, v.MapRequiredKey[namedMap]("k0", order, v.NotEmpty[string]()))
	}
	return stored(v.All(rules...), input)
}

func integerKeyMap() func() error {
	input := make(namedIntMap, 64)
	for i := 1; i <= 64; i++ {
		input[i] = namedString("ok")
	}
	order := v.KeyOrder[int]{
		Less: func(left, right int) bool { return left < right },
		Text: strconv.Itoa,
	}
	rule := v.All(
		v.MapValues[namedIntMap](order, v.NotEmpty[namedString]()),
		v.MapKeys[namedIntMap](order, v.Positive[int]()),
		v.MapEach[namedIntMap](order, v.Project(func(entry v.Entry[int, namedString]) namedString { return entry.Value }, v.NotEmpty[namedString]())),
	)
	return stored(rule, input)
}

func scalarCheck() func() error {
	rule := v.Check(func(value int) bool { return value == 7 || value == 9 }, func(int) error { return v.NewViolation(v.CodeEqual, nil) })
	return stored(rule, 9)
}

func controls(name string) func() error {
	switch name {
	case "control-blank":
		return func() error { return nil }
	case "control-first":
		first := true
		return func() error {
			if first {
				retained = make([]byte, 1)
				first = false
			}
			return nil
		}
	case "control-rare":
		calls := 0
		return func() error {
			calls++
			if calls%100 == 0 {
				retained = make([]byte, 1)
			}
			return nil
		}
	case "control-callback":
		return func() error {
			retained = make([]byte, 1)
			return nil
		}
	default:
		panic("unknown control fixture " + name)
	}
}

var fixtureCatalog = []fixtureInfo{
	{Name: "scalar-signed", Kind: allocationFree},
	{Name: "scalar-extremes", Kind: allocationFree},
	{Name: "scalar-zero", Kind: allocationFree},
	{Name: "scalar-negative", Kind: allocationFree},
	{Name: "scalar-unsigned-named", Kind: allocationFree},
	{Name: "scalar-float", Kind: allocationFree},
	{Name: "scalar-float-infinity", Kind: allocationFree},
	{Name: "scalar-float-negative-zero", Kind: allocationFree},
	{Name: "scalar-comparator", Kind: allocationFree},
	{Name: "scalar-comparable", Kind: allocationFree},
	{Name: "strings-named", Kind: allocationFree},
	{Name: "bytes-named", Kind: allocationFree},
	{Name: "time", Kind: allocationFree},
	{Name: "fields-flat-16", Kind: allocationFree},
	{Name: "fields-flat-1", Kind: allocationFree},
	{Name: "struct-conditions", Kind: allocationFree},
	{Name: "fields-deep-1", Kind: allocationFree},
	{Name: "fields-deep-4", Kind: allocationFree},
	{Name: "fields-deep-16", Kind: allocationFree},
	{Name: "presence-optional-absent", Kind: allocationFree},
	{Name: "presence-required-present", Kind: allocationFree},
	{Name: "presence-value-absent", Kind: allocationFree},
	{Name: "presence-value-present", Kind: allocationFree},
	{Name: "presence-value-present-zero", Kind: allocationFree},
	{Name: "presence-nested-pointer", Kind: allocationFree},
	{Name: "array-pointer-project", Kind: allocationFree},
	{Name: "slice-size-0", Kind: allocationFree},
	{Name: "slice-size-1", Kind: allocationFree},
	{Name: "slice-size-8", Kind: allocationFree},
	{Name: "slice-size-64", Kind: allocationFree},
	{Name: "slice-size-1024", Kind: allocationFree},
	{Name: "slice-nested-named", Kind: allocationFree},
	{Name: "slice-structs", Kind: allocationFree},
	{Name: "membership-size-0", Kind: allocationFree},
	{Name: "membership-size-1", Kind: allocationFree},
	{Name: "membership-size-4", Kind: allocationFree},
	{Name: "membership-size-32", Kind: allocationFree},
	{Name: "membership-size-1024", Kind: allocationFree},
	{Name: "map-size-0", Kind: allocationFree},
	{Name: "map-size-1", Kind: allocationFree},
	{Name: "map-size-8", Kind: allocationFree},
	{Name: "map-size-64", Kind: allocationFree},
	{Name: "map-size-1024", Kind: allocationFree},
	{Name: "map-size-10000", Kind: allocationFree},
	{Name: "map-typed-integer-keys", Kind: allocationFree},
	{Name: "check-last-alternative", Kind: allocationFree},
	{Name: "generic-methods", Kind: allocationFree},
	{Name: "control-blank", Kind: control},
	{Name: "control-first", Kind: control},
	{Name: "control-rare", Kind: control},
	{Name: "control-callback", Kind: control},
}

func fixture(name string) preparedFixture {
	for _, info := range fixtureCatalog {
		if info.Name != name {
			continue
		}
		var run func() error
		switch name {
		case "scalar-signed":
			run = scalarSigned()
		case "scalar-extremes":
			run = scalarExtremes()
		case "scalar-zero":
			run = scalarZero()
		case "scalar-negative":
			run = scalarNegative()
		case "scalar-unsigned-named":
			run = scalarUnsigned()
		case "scalar-float":
			run = scalarFloat()
		case "scalar-float-infinity":
			run = scalarInfinity()
		case "scalar-float-negative-zero":
			run = scalarSignedZero()
		case "scalar-comparator":
			run = comparator()
		case "scalar-comparable":
			run = scalarComparable()
		case "strings-named":
			run = stringCatalogue()
		case "bytes-named":
			run = bytesCatalogue()
		case "time":
			run = timeCatalogue()
		case "fields-flat-16":
			run = flatFields()
		case "fields-flat-1":
			run = flatField1()
		case "struct-conditions":
			run = structConditions()
		case "fields-deep-1":
			run = deepFields(1)
		case "fields-deep-4":
			run = deepFields(4)
		case "fields-deep-16":
			run = deepFields(16)
		case "presence-optional-absent":
			run = presenceOptionalAbsent()
		case "presence-required-present":
			run = presenceRequiredPresent()
		case "presence-value-absent":
			run = presenceValueAbsent()
		case "presence-value-present":
			run = presenceValuePresent()
		case "presence-value-present-zero":
			run = presenceValuePresentZero()
		case "presence-nested-pointer":
			run = presenceNested()
		case "array-pointer-project":
			run = arrayProjection()
		case "slice-size-0":
			run = sliceBySize(0)
		case "slice-size-1":
			run = sliceBySize(1)
		case "slice-size-8":
			run = sliceBySize(8)
		case "slice-size-64":
			run = sliceBySize(64)
		case "slice-size-1024":
			run = sliceBySize(1024)
		case "slice-nested-named":
			run = namedNestedSlices()
		case "slice-structs":
			run = sliceOfStructs()
		case "membership-size-0":
			run = membership(0)
		case "membership-size-1":
			run = membership(1)
		case "membership-size-4":
			run = membership(4)
		case "membership-size-32":
			run = membership(32)
		case "membership-size-1024":
			run = membership(1024)
		case "map-size-0":
			run = mapRule(0)
		case "map-size-1":
			run = mapRule(1)
		case "map-size-8":
			run = mapRule(8)
		case "map-size-64":
			run = mapRule(64)
		case "map-size-1024":
			run = mapRule(1024)
		case "map-size-10000":
			run = mapRule(10000)
		case "map-typed-integer-keys":
			run = integerKeyMap()
		case "check-last-alternative":
			run = scalarCheck()
		case "generic-methods":
			run = genericMethods()
		default:
			run = controls(name)
		}
		return preparedFixture{fixtureInfo: info, run: run}
	}
	fmt.Fprintln(os.Stderr, "unknown fixture:", name)
	os.Exit(2)
	return preparedFixture{}
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "list" {
		if err := json.NewEncoder(os.Stdout).Encode(fixtureCatalog); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: allocprobe list | FIXTURE first|batch")
		os.Exit(2)
	}
	phase := os.Args[2]
	if phase != "first" && phase != "batch" {
		fmt.Fprintln(os.Stderr, "invalid phase:", phase)
		os.Exit(2)
	}
	prepared := fixture(os.Args[1])

	// Disable collection after construction. The GC and runtime metadata
	// snapshots are outside the measured region.
	debug.SetGCPercent(-1)
	runtime.GC()
	calls := 1
	if phase == "batch" {
		calls = 1000
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range calls {
		if err := prepared.run(); err != nil {
			fmt.Fprintln(os.Stderr, "fixture returned an error:", err)
			os.Exit(1)
		}
	}
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(retained)
	result := report{
		Fixture: prepared.Name,
		Kind:    prepared.Kind,
		Phase:   phase,
		Calls:   calls,
		Mallocs: after.Mallocs - before.Mallocs,
		Bytes:   after.TotalAlloc - before.TotalAlloc,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
