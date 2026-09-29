package validation

import (
	"iter"
	"strconv"
	"strings"
)

// Code identifies a validation failure independently of its message.
type Code string

const (
	CodeExternal          Code = "external"
	CodeRequired          Code = "required"
	CodeKeyRequired       Code = "key_required"
	CodeIndexOutOfRange   Code = "index_out_of_range"
	CodeEqual             Code = "equal"
	CodeNotEqual          Code = "not_equal"
	CodeZero              Code = "zero"
	CodeNotZero           Code = "not_zero"
	CodeOneOf             Code = "one_of"
	CodeNotOneOf          Code = "not_one_of"
	CodeMin               Code = "min"
	CodeMax               Code = "max"
	CodeBetween           Code = "between"
	CodeGreaterThan       Code = "greater_than"
	CodeLessThan          Code = "less_than"
	CodePositive          Code = "positive"
	CodeNonNegative       Code = "non_negative"
	CodeNegative          Code = "negative"
	CodeNonPositive       Code = "non_positive"
	CodeNotNaN            Code = "not_nan"
	CodeFinite            Code = "finite"
	CodeNotEmpty          Code = "not_empty"
	CodeByteLength        Code = "byte_length"
	CodeByteMinLength     Code = "byte_min_length"
	CodeByteMaxLength     Code = "byte_max_length"
	CodeByteLengthBetween Code = "byte_length_between"
	CodeRuneLength        Code = "rune_length"
	CodeRuneMinLength     Code = "rune_min_length"
	CodeRuneMaxLength     Code = "rune_max_length"
	CodeRuneLengthBetween Code = "rune_length_between"
	CodeContains          Code = "contains"
	CodePrefix            Code = "prefix"
	CodeSuffix            Code = "suffix"
	CodeUTF8              Code = "utf8"
	CodeLength            Code = "length"
	CodeMinLength         Code = "min_length"
	CodeMaxLength         Code = "max_length"
	CodeLengthBetween     Code = "length_between"
	CodeUnique            Code = "unique"
	CodeBefore            Code = "before"
	CodeBeforeOrEqual     Code = "before_or_equal"
	CodeAfter             Code = "after"
	CodeAfterOrEqual      Code = "after_or_equal"
)

// Coded is an error with a stable machine-readable failure code.
type Coded interface {
	error
	Code() Code
}

// ConfigurationError reports invalid static rule configuration.
type ConfigurationError struct{ constructor string }

func (e *ConfigurationError) Constructor() string { return e.constructor }
func (e *ConfigurationError) Error() string       { return "invalid configuration for " + e.constructor }
func configurationError(constructor string)       { panic(&ConfigurationError{constructor}) }

// Violation is a coded failure with an optional ordinary cause.
type Violation struct {
	code  Code
	cause error
}

func NewViolation(code Code, cause error) *Violation {
	if code == "" {
		configurationError("NewViolation")
	}
	return &Violation{code: code, cause: cause}
}
func (e *Violation) Code() Code    { return e.code }
func (e *Violation) Error() string { return Format(e) }
func (e *Violation) Unwrap() error { return e.cause }

type LengthUnit string

const (
	LengthBytes    LengthUnit = "bytes"
	LengthRunes    LengthUnit = "runes"
	LengthElements LengthUnit = "elements"
	LengthEntries  LengthUnit = "entries"
)

// LengthError describes a failed length constraint.
type LengthError struct {
	code                     Code
	actual, minimum, maximum int
	hasMaximum               bool
	unit                     LengthUnit
}

func (e *LengthError) Code() Code           { return e.code }
func (e *LengthError) Error() string        { return Format(e) }
func (e *LengthError) Actual() int          { return e.actual }
func (e *LengthError) Minimum() int         { return e.minimum }
func (e *LengthError) Maximum() (int, bool) { return e.maximum, e.hasMaximum }
func (e *LengthError) Unit() LengthUnit     { return e.unit }
func newLengthError(code Code, actual, minimum int, maximum *int, unit LengthUnit) error {
	e := &LengthError{code: code, actual: actual, minimum: minimum, unit: unit}
	if maximum != nil {
		e.maximum, e.hasMaximum = *maximum, true
	}
	return e
}

type bound[T any] struct {
	value     T
	inclusive bool
}

// BoundsError describes a failed typed range or comparison constraint.
type BoundsError[T any] struct {
	code               Code
	lower, upper       bound[T]
	hasLower, hasUpper bool
}

func (e *BoundsError[T]) Code() Code             { return e.code }
func (e *BoundsError[T]) Error() string          { return Format(e) }
func (e *BoundsError[T]) Lower() (T, bool, bool) { return e.lower.value, e.lower.inclusive, e.hasLower }
func (e *BoundsError[T]) Upper() (T, bool, bool) { return e.upper.value, e.upper.inclusive, e.hasUpper }
func newBoundsError[T any](code Code, lower *bound[T], upper *bound[T]) error {
	e := &BoundsError[T]{code: code}
	if lower != nil {
		e.lower, e.hasLower = *lower, true
	}
	if upper != nil {
		e.upper, e.hasUpper = *upper, true
	}
	return e
}

// DuplicateError identifies the earliest prior equal element.
type DuplicateError struct{ first int }

func (e *DuplicateError) Code() Code      { return CodeUnique }
func (e *DuplicateError) Error() string   { return Format(e) }
func (e *DuplicateError) FirstIndex() int { return e.first }
func newDuplicateError(first int) error   { return &DuplicateError{first} }

// IndexError describes an absent addressed element.
type IndexError struct{ index, length int }

func (e *IndexError) Code() Code            { return CodeIndexOutOfRange }
func (e *IndexError) Error() string         { return Format(e) }
func (e *IndexError) Index() int            { return e.index }
func (e *IndexError) Length() int           { return e.length }
func newIndexError(index, length int) error { return &IndexError{index, length} }

type SegmentKind uint8

const (
	FieldSegment SegmentKind = iota + 1
	IndexSegment
	KeySegment
)

type Segment struct {
	Kind  SegmentKind
	Name  string
	Index int
}
type Issue struct {
	Path []Segment
	Code Code
	Err  error
}

type located struct {
	segment Segment
	child   error
}

func (e *located) Error() string { return Format(e) }
func (e *located) Unwrap() error { return e.child }
func at(segment Segment, err error) error {
	if err == nil {
		return nil
	}
	validateSegment(segment, "location")
	return &located{segment, err}
}

type aggregate struct{ children []error }

func (e *aggregate) Error() string   { return Format(e) }
func (e *aggregate) Unwrap() []error { return append([]error(nil), e.children...) }
func combine(children []error) error {
	var first error
	count := 0
	for _, child := range children {
		if child != nil {
			if count == 0 {
				first = child
			}
			count++
		}
	}
	switch count {
	case 0:
		return nil
	case 1:
		return first
	default:
		owned := make([]error, 0, count)
		for _, child := range children {
			if child != nil {
				owned = append(owned, child)
			}
		}
		return &aggregate{children: owned}
	}
}
func appendFailure(children []error, err error) []error {
	if err != nil {
		return append(children, err)
	}
	return children
}

// Issues returns one independent, owned path snapshot per failure occurrence.
func Issues(err error) []Issue {
	var out []Issue
	for issue := range walkIssues(err) {
		out = append(out, issue)
	}
	return out
}

func walkIssues(err error) iter.Seq[Issue] {
	return func(yield func(Issue) bool) {
		walkIssueNode(err, nil, yield)
	}
}
func walkIssueNode(err error, path []Segment, yield func(Issue) bool) bool {
	if err == nil {
		return true
	}
	if l, ok := err.(*located); ok {
		return walkIssueNode(l.child, append(path, l.segment), yield)
	}
	if coded, ok := err.(Coded); ok {
		return yield(Issue{Path: append([]Segment(nil), path...), Code: coded.Code(), Err: err})
	}
	if multiple, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range multiple.Unwrap() {
			if !walkIssueNode(child, path, yield) {
				return false
			}
		}
		return true
	}
	if single, ok := err.(interface{ Unwrap() error }); ok {
		if child := single.Unwrap(); child != nil {
			return walkIssueNode(child, path, yield)
		}
	}
	return yield(Issue{Path: append([]Segment(nil), path...), Code: CodeExternal, Err: err})
}

// FormatPath renders a typed path without conflating fields, indices, and keys.
func FormatPath(path []Segment) string {
	var b strings.Builder
	b.WriteByte('$')
	for _, segment := range path {
		validateSegment(segment, "FormatPath")
		switch segment.Kind {
		case FieldSegment:
			if identifier(segment.Name) {
				b.WriteByte('.')
				b.WriteString(segment.Name)
			} else {
				b.WriteString(".[")
				b.WriteString(strconv.Quote(segment.Name))
				b.WriteByte(']')
			}
		case IndexSegment:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(segment.Index))
			b.WriteByte(']')
		case KeySegment:
			b.WriteByte('[')
			b.WriteString(strconv.Quote(segment.Name))
			b.WriteByte(']')
		}
	}
	return b.String()
}
func validateSegment(s Segment, constructor string) {
	switch s.Kind {
	case FieldSegment:
		if s.Name == "" {
			configurationError(constructor)
		}
	case IndexSegment:
		if s.Index < 0 {
			configurationError(constructor)
		}
	case KeySegment:
	default:
		configurationError(constructor)
	}
}
func identifier(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			continue
		}
		if i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

// Format presents paths and codes without invoking external Error methods.
func Format(err error) string {
	var b strings.Builder
	for issue := range walkIssues(err) {
		if b.Len() > 0 {
			b.WriteString("; ")
		}
		b.WriteString(FormatPath(issue.Path))
		b.WriteString(": ")
		b.WriteString(string(issue.Code))
	}
	return b.String()
}
