package validation

import "slices"

// Entry is the typed value supplied by MapEach for one map entry.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

// KeyOrder defines ordering and canonical path text for a supported key type.
type KeyOrder[K comparable] struct {
	Less func(K, K) bool
	Text func(K) string
}

// StringKeys returns lexicographic ordering and exact text for string-like keys.
func StringKeys[K ~string]() KeyOrder[K] {
	return KeyOrder[K]{
		Less: func(left, right K) bool { return left < right },
		Text: func(key K) string { return string(key) },
	}
}

// MapLength requires a map to contain exactly n entries.
func MapLength[M ~map[K]V, K comparable, V any](n int) Rule[M] {
	if n < 0 {
		configurationError("MapLength")
	}
	return func(values M) error {
		if len(values) == n {
			return nil
		}
		return newLengthError(CodeLength, len(values), n, &n, LengthEntries)
	}
}

// MapMinLength requires a map to contain at least min entries.
func MapMinLength[M ~map[K]V, K comparable, V any](min int) Rule[M] {
	if min < 0 {
		configurationError("MapMinLength")
	}
	return func(values M) error {
		if len(values) >= min {
			return nil
		}
		return newLengthError(CodeMinLength, len(values), min, nil, LengthEntries)
	}
}

// MapMaxLength requires a map to contain at most max entries.
func MapMaxLength[M ~map[K]V, K comparable, V any](max int) Rule[M] {
	if max < 0 {
		configurationError("MapMaxLength")
	}
	return func(values M) error {
		if len(values) <= max {
			return nil
		}
		return newLengthError(CodeMaxLength, len(values), 0, &max, LengthEntries)
	}
}

// MapLengthBetween requires a map length in the inclusive interval [min, max].
func MapLengthBetween[M ~map[K]V, K comparable, V any](min, max int) Rule[M] {
	if min < 0 || max < 0 || min > max {
		configurationError("MapLengthBetween")
	}
	return func(values M) error {
		if len(values) >= min && len(values) <= max {
			return nil
		}
		return newLengthError(CodeLengthBetween, len(values), min, &max, LengthEntries)
	}
}

// MapEach applies every child rule to every typed entry. Failing entries are
// ordered by order after validation; successful entries do not invoke order.
func MapEach[M ~map[K]V, K comparable, V any](order KeyOrder[K], rules ...Rule[Entry[K, V]]) Rule[M] {
	order = checkedKeyOrder(order, "MapEach")
	children := checkRules("MapEach", rules)
	return func(values M) error {
		var failures []mapFailure[K]
		for key, value := range values {
			var entryFailures []error
			entry := Entry[K, V]{Key: key, Value: value}
			for _, rule := range children {
				entryFailures = appendFailure(entryFailures, rule(entry))
			}
			if err := combine(entryFailures); err != nil {
				failures = append(failures, mapFailure[K]{key: key, err: err})
			}
		}
		return finishMapFailures(order, failures)
	}
}

// MapKeys applies every child rule to every key.
func MapKeys[M ~map[K]V, K comparable, V any](order KeyOrder[K], rules ...Rule[K]) Rule[M] {
	order = checkedKeyOrder(order, "MapKeys")
	children := checkRules("MapKeys", rules)
	return func(values M) error {
		var failures []mapFailure[K]
		for key := range values {
			var keyFailures []error
			for _, rule := range children {
				keyFailures = appendFailure(keyFailures, rule(key))
			}
			if err := combine(keyFailures); err != nil {
				failures = append(failures, mapFailure[K]{key: key, err: err})
			}
		}
		return finishMapFailures(order, failures)
	}
}

// MapValues applies every child rule to every value.
func MapValues[M ~map[K]V, K comparable, V any](order KeyOrder[K], rules ...Rule[V]) Rule[M] {
	order = checkedKeyOrder(order, "MapValues")
	children := checkRules("MapValues", rules)
	return func(values M) error {
		var failures []mapFailure[K]
		for key, value := range values {
			var valueFailures []error
			for _, rule := range children {
				valueFailures = appendFailure(valueFailures, rule(value))
			}
			if err := combine(valueFailures); err != nil {
				failures = append(failures, mapFailure[K]{key: key, err: err})
			}
		}
		return finishMapFailures(order, failures)
	}
}

// MapRequiredKey validates a present key and reports key_required when absent.
func MapRequiredKey[M ~map[K]V, K comparable, V any](key K, order KeyOrder[K], rules ...Rule[V]) Rule[M] {
	order = checkedKeyOrder(order, "MapRequiredKey")
	children := checkRules("MapRequiredKey", rules)
	return func(values M) error {
		value, ok := values[key]
		if !ok {
			return at(Segment{Kind: KeySegment, Name: order.Text(key)}, NewViolation(CodeKeyRequired, nil))
		}
		var failures []error
		for _, rule := range children {
			failures = appendFailure(failures, rule(value))
		}
		if err := combine(failures); err != nil {
			return at(Segment{Kind: KeySegment, Name: order.Text(key)}, err)
		}
		return nil
	}
}

// MapOptionalKey validates a present key and succeeds when absent.
func MapOptionalKey[M ~map[K]V, K comparable, V any](key K, order KeyOrder[K], rules ...Rule[V]) Rule[M] {
	order = checkedKeyOrder(order, "MapOptionalKey")
	children := checkRules("MapOptionalKey", rules)
	return func(values M) error {
		value, ok := values[key]
		if !ok {
			return nil
		}
		var failures []error
		for _, rule := range children {
			failures = appendFailure(failures, rule(value))
		}
		if err := combine(failures); err != nil {
			return at(Segment{Kind: KeySegment, Name: order.Text(key)}, err)
		}
		return nil
	}
}

// MapKeysOneOf requires every map key to be in allowed.
func MapKeysOneOf[M ~map[K]V, K comparable, V any](order KeyOrder[K], allowed ...K) Rule[M] {
	order = checkedKeyOrder(order, "MapKeysOneOf")
	set := make(map[K]struct{}, len(allowed))
	for _, key := range allowed {
		set[key] = struct{}{}
	}
	return func(values M) error {
		var failures []mapFailure[K]
		for key := range values {
			if _, ok := set[key]; ok {
				continue
			}
			failures = append(failures, mapFailure[K]{key: key, err: NewViolation(CodeOneOf, nil)})
		}
		return finishMapFailures(order, failures)
	}
}

// MapKeysNotOneOf rejects every map key in forbidden.
func MapKeysNotOneOf[M ~map[K]V, K comparable, V any](order KeyOrder[K], forbidden ...K) Rule[M] {
	order = checkedKeyOrder(order, "MapKeysNotOneOf")
	set := make(map[K]struct{}, len(forbidden))
	for _, key := range forbidden {
		set[key] = struct{}{}
	}
	return func(values M) error {
		var failures []mapFailure[K]
		for key := range values {
			if _, ok := set[key]; !ok {
				continue
			}
			failures = append(failures, mapFailure[K]{key: key, err: NewViolation(CodeNotOneOf, nil)})
		}
		return finishMapFailures(order, failures)
	}
}

type mapFailure[K comparable] struct {
	key K
	err error
}

func checkedKeyOrder[K comparable](order KeyOrder[K], constructor string) KeyOrder[K] {
	if order.Less == nil || order.Text == nil {
		configurationError(constructor)
	}
	return order
}

func finishMapFailures[K comparable](order KeyOrder[K], failures []mapFailure[K]) error {
	if len(failures) == 0 {
		return nil
	}
	slices.SortFunc(failures, func(left, right mapFailure[K]) int {
		if order.Less(left.key, right.key) {
			return -1
		}
		if order.Less(right.key, left.key) {
			return 1
		}
		return 0
	})
	var result []error
	for _, failure := range failures {
		result = appendFailure(result, at(Segment{
			Kind: KeySegment,
			Name: order.Text(failure.key),
		}, failure.err))
	}
	return combine(result)
}
