package validation_test

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	v "github.com/jacoelho/validation"
)

func TestConcurrentRuleReuse(t *testing.T) {
	type Input struct {
		Name   string
		Values []int
		Labels map[string]int
		State  string
	}
	shared := errors.New("shared state failure")
	rule := v.All(
		v.Field("name", func(i Input) string { return i.Name }, v.NotEmpty[string](), v.RuneMinLength[string](2)),
		v.Field("values", func(i Input) []int { return i.Values }, v.Each[[]int](v.Positive[int]())),
		v.Field("labels", func(i Input) map[string]int { return i.Labels }, v.MapValues[map[string]int](v.StringKeys[string](), v.Positive[int]())),
		v.Field("state", func(i Input) string { return i.State }, v.Rule[string](func(state string) error {
			if state == "bad" {
				return shared
			}
			return nil
		})),
	)
	valid := Input{Name: "good", Values: []int{1, 2, 3}, Labels: map[string]int{"a": 1, "b": 2}, State: "ok"}
	invalid := Input{Name: "", Values: []int{0, -1}, Labels: map[string]int{"b": 0, "a": -1}, State: "bad"}
	want := "$.name: not_empty; $.name: rune_min_length; $.values[0]: positive; $.values[1]: positive; $.labels[\"a\"]: positive; $.labels[\"b\"]: positive; $.state: external"
	before := map[string]int{"b": 0, "a": -1}
	saved := rule(invalid)
	failures := make(chan string, 32)
	var wg sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if err := rule(valid); err != nil {
					failures <- fmt.Sprintf("valid = %s", v.Format(err))
					return
				}
				err := rule(invalid)
				if got := v.Format(err); got != want || !errors.Is(err, shared) {
					failures <- fmt.Sprintf("invalid = %s, shared identity=%t", got, errors.Is(err, shared))
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	if got := v.Format(saved); got != want || !errors.Is(saved, shared) {
		t.Fatalf("saved result changed: %s", got)
	}
	if valid.Name != "good" || invalid.Name != "" || !reflect.DeepEqual(invalid.Values, []int{0, -1}) || !reflect.DeepEqual(invalid.Labels, before) {
		t.Fatal("input changed")
	}
}
