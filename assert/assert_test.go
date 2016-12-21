/*
Copyright 2017 Turbine Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package assert

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type complexStruct struct {
	x int
	y *string
}

type equalityKind int

const (
	notEqual equalityKind = iota
	justEqual
	justDeepEqual
	justJsonEqual
	equalAndDeepEqual
)

type nilnessKind bool

const (
	notNilish nilnessKind = false
	nilish                = true
)

type equalTestCase struct {
	name string
	a    interface{}
	b    interface{}
	kind equalityKind
}

func (e equalTestCase) run(
	t *testing.T,
	f func(testing.TB, interface{}, interface{}) bool,
	expectEqual func(equalTestCase) bool,
) {
	tr := Tracing(t)

	defer func() {
		if p := recover(); p != nil {
			tr.Errorf("%s: panic: %v", e.name, p)
		}
	}()

	testT := &testing.T{}

	result := f(testT, e.a, e.b)
	expectedResult := expectEqual(e)

	if result != expectedResult {
		comparison := "equal"
		if !expectedResult {
			comparison = "not equal"
		}
		tr.Errorf("%s: expected %+v to %s %+v", e.name, e.a, comparison, e.b)
	}
}

type nilTestCase struct {
	name string
	v    interface{}
	kind nilnessKind
}

func (n nilTestCase) run(
	t *testing.T,
	f func(testing.TB, interface{}) bool,
	expectNil func(nilTestCase) bool,
) {
	tr := Tracing(t)
	testT := &testing.T{}

	result := f(testT, n.v)
	expectedResult := expectNil(n)

	if result != expectedResult {
		tr.Errorf("%s: expected %t, got %t for %+v", n.name, expectedResult, result, n.v)
	}
}

type logEntry struct {
	op   string
	args string
}

type mockT struct {
	testing.TB

	log []logEntry
}

func (t *mockT) record(op string, args string) {
	entry := logEntry{op, args}
	if t.log == nil {
		t.log = make([]logEntry, 1)
		t.log[0] = entry
	} else {
		t.log = append(t.log, entry)
	}
}

func (t *mockT) reset() {
	t.log = nil
}

func (t *mockT) checkErrorPrefix(realT testing.TB, prefix string) {
	if len(t.log) != 1 {
		realT.Errorf("expected single error, got '%+v'", t.log)
	}

	switch t.log[0].op {
	case "Error", "Errorf":
		if !strings.HasPrefix(t.log[0].args, prefix) {
			realT.Errorf("got %q, expected prefix %q", t.log[0].args, prefix)
		}
	default:
		realT.Errorf("expected Error or Error op, got '%+v'", t.log[0])
	}
}

// For any testing.TB method invoked on mockT, you'll need to
// override the version inherited from embedding a testing.TB in
// mockT. (The embedded versions of the methods will fail due to the
// TB field being nil.)
func (t *mockT) Errorf(format string, args ...interface{}) {
	t.record("Errorf", fmt.Sprintf(format, args...))
}

func (t *mockT) Error(args ...interface{}) {
	t.record("Error", strings.TrimRight(fmt.Sprintln(args...), "\n"))
}

func (t *mockT) Fatalf(format string, args ...interface{}) {
	t.record("Fatalf", fmt.Sprintf(format, args...))
}

func (t *mockT) Fatal(args ...interface{}) {
	t.record("Fatal", strings.TrimRight(fmt.Sprintln(args...), "\n"))
}

type moreComplexStruct struct {
	A string               `json:"a"`
	C lessComplexSubstruct `json:"c"`
}
type lessComplexSubstruct struct {
	D string `json:"d"`
}

var (
	nilStringPtr *string   = nil
	nilStructPtr *struct{} = nil
	nilChannel   chan<- bool
	nilFunction  func()
	nilSlice     []string

	channel  = make(chan<- bool)
	function = func() {}

	int1a = 123
	int1b = 123
	int2  = 456

	string1a = "string"
	string1b = "string"
	string2  = "other string"

	cs1a = complexStruct{1, &string1a}
	cs1b = complexStruct{1, &string1a}
	cs2a = complexStruct{1, &string1a}
	cs2b = complexStruct{1, &string1b}
	cs3  = complexStruct{1, &string1a}
	cs4  = complexStruct{1, &string2}

	array1a = [3]string{"a", "b", "c"}
	array1b = [3]string{"a", "b", "c"}
	array2  = [3]string{"X", "Y", "Z"}

	slice1a     = []string{"a", "b", "c"}
	slice1b     = []string{"a", "b", "c"}
	slice2      = []string{"X", "Y", "Z"}
	slice2trunc = slice2[0:2]

	map1    = map[string]interface{}{"a": "b", "c": map[string]string{"d": "e"}}
	struct1 = moreComplexStruct{A: "b", C: lessComplexSubstruct{D: "e"}}
	map2    = map[string]interface{}{"a": "b", "c": map[string]string{"d": "z"}}
	struct2 = moreComplexStruct{A: "b", C: lessComplexSubstruct{D: "z"}}

	iface1 interface{} = new(interface{})
	iface2 interface{} = new(interface{})

	nilnessTestCases = []nilTestCase{
		{"nil", nil, nilish},
		{"*string-nil", nilStringPtr, nilish},
		{"*struct-nil", nilStructPtr, nilish},
		{"*int", &int1a, notNilish},
		{"*string-1a", &string1a, notNilish},
		{"*struct-1a", &cs1a, notNilish},
		{"[]string-nilish", nilSlice, nilish},
		{"[]string-notnilish", slice1a, notNilish},
		{"[n]string-notnilish", array1a, notNilish},
		{"chan-nilish", nilChannel, nilish},
		{"chan-notnilish", channel, notNilish},
		{"func-nilish", nilFunction, nilish},
		{"func-notnilish", function, notNilish},
	}

	valueEqualityTestCasesJsonOk = []equalTestCase{
		{"string-1a-1b", string1a, string1b, equalAndDeepEqual},
		{"string-1a-2", string1a, string2, notEqual},
		{"int-1a-1b", int1a, int1b, equalAndDeepEqual},
		{"int-1a-2", int1a, int2, notEqual},
		{"struct-1a-1b", cs1a, cs1b, equalAndDeepEqual},
		{"struct-2a-2b", cs2a, cs2b, justDeepEqual},
		{"struct-3-4", cs3, cs4, justJsonEqual},
	}

	pointerEqualityTestCasesJsonOk = []equalTestCase{
		{"*string-nil-nil", nilStringPtr, nilStringPtr, equalAndDeepEqual},
		{"*string-1a-1b", &string1a, &string1b, justDeepEqual},
		{"*string-1a-2", &string1a, &string2, notEqual},
		{"string & *-1a-1b", string1a, &string1b, justJsonEqual},
		{"string & *-1a-1b", &string1a, string1b, justJsonEqual},
		{"*int-1a-1b", &int1a, &int1b, justDeepEqual},
		{"*int-1a-2", &int1a, &int2, notEqual},
		{"*struct-1a-1b", &cs1a, &cs1b, justDeepEqual},
		{"*struct-2a-2b", &cs2a, &cs2b, justDeepEqual},
		{"*struct-3-4", &cs3, &cs4, justJsonEqual},
		{"*[]string-1a-1b", &slice1a, &slice1b, justDeepEqual},
		{"*[]string-1a-2", &slice1a, &slice2, notEqual},
	}
	arrayPointerEqualityTestCasesJsonOk = []equalTestCase{
		{"*[n]string-1a-1b", &array1a, &array1b, justDeepEqual},
		{"*[n]string-1a-2", &array1a, &array2, notEqual},
	}

	equalityTestCasesNoJson = []equalTestCase{
		// these types cannot be json marshalled
		{"chan", channel, channel, equalAndDeepEqual},
	}

	equalityTestCasesJsonOk = append(
		valueEqualityTestCasesJsonOk,
		append(
			pointerEqualityTestCasesJsonOk,
			arrayPointerEqualityTestCasesJsonOk...,
		)...,
	)

	equalityTestCases = append(
		equalityTestCasesJsonOk,
		equalityTestCasesNoJson...,
	)

	pointerDeepEqualityTestCasesJsonOk = []equalTestCase{
		{"[]string-1a-1b", slice1a, slice1b, justDeepEqual},
		{"[]string-1a-2", slice1a, slice2, notEqual},
		{"[]string-2-2trunc", slice2, slice2trunc, notEqual},
	}

	arrayDeepEqualityTestCasesJsonOk = []equalTestCase{
		{"[n]string-1a-1b", array1a, array1b, justDeepEqual},
		{"[n]string-1a-2", array1a, array2, notEqual},
	}

	// these types cannot be compared with == (runtime panic)
	deepEqualityTestCasesJsonOk = append(
		pointerDeepEqualityTestCasesJsonOk,
		arrayDeepEqualityTestCasesJsonOk...,
	)

	deepEqualityTestCases = append(
		deepEqualityTestCasesJsonOk,
		// these types cannot be json marshalled
		equalTestCase{"func", function, function, justEqual},
	)

	pointerDeepEqualityTestCases = append(
		pointerDeepEqualityTestCasesJsonOk,
		// these types cannot be json marshalled
		equalTestCase{"func", function, function, justEqual},
	)

	justJsonEqualTestCases = []equalTestCase{
		{"map1a-struct", map1, struct1, justJsonEqual},
		{"map1a-map2", map1, map2, notEqual},
		{"map2-struct", map2, struct1, notEqual},
		{"struct1-cs3", struct1, cs3, notEqual},
		{"struct1-struct2", struct1, struct2, notEqual},
	}

	sameInstanceTestCases = append(
		pointerEqualityTestCasesJsonOk,
		append(
			pointerDeepEqualityTestCases,
			[]equalTestCase{
				{"map1a-struct", map1, struct1, justJsonEqual},
				{"map1a-map2", map1, map2, notEqual},
				{"map2-struct", map2, struct1, notEqual},
				{"iface1-iface1", iface1, iface1, justEqual},
				{"iface1-iface2", iface1, iface2, notEqual},
			}...,
		)...,
	)
)

func TestTracing(t *testing.T) {
	switch obj := Tracing(t).(type) {
	case *TracingTB:
		Equal(obj, obj.TB, t)
	default:
		obj.Errorf("got *TracingTB, want %T", obj)
	}
}

func TestTracingNoWrap(t *testing.T) {
	tr := Tracing(t)
	obj := Tracing(tr)
	Equal(tr, tr, obj)
}

func TestTracingNoWrapG(t *testing.T) {
	Group("Foo", t, func(g *G) {
		obj := Tracing(g)
		Equal(g, g, obj)
	})
}

func TestNonNil(t *testing.T) {
	for _, test := range nilnessTestCases {
		test.run(
			t,
			NonNil,
			func(test nilTestCase) bool {
				return test.kind == notNilish
			})
	}
}

func TestNil(t *testing.T) {
	for _, test := range nilnessTestCases {
		test.run(
			t,
			Nil,
			func(test nilTestCase) bool {
				return test.kind == nilish
			})
	}
}

func TestEqual(t *testing.T) {
	for _, test := range equalityTestCases {
		test.run(
			t,
			Equal,
			func(test equalTestCase) bool {
				return test.kind == justEqual || test.kind == equalAndDeepEqual
			})
	}
}

func TestNotEqual(t *testing.T) {
	for _, test := range equalityTestCases {
		test.run(
			t,
			NotEqual,
			func(test equalTestCase) bool {
				return test.kind == notEqual || test.kind == justDeepEqual || test.kind == justJsonEqual
			})
	}
}

func TestDeepEqual(t *testing.T) {
	testCases := append(equalityTestCases, deepEqualityTestCases...)
	for _, test := range testCases {
		test.run(
			t,
			DeepEqual,
			func(test equalTestCase) bool {
				return test.kind == justDeepEqual || test.kind == equalAndDeepEqual
			})
	}
}

func TestArrayEqual(t *testing.T) {
	// slices
	var nilSlice []string
	emptySlice := []string{}
	s1 := []string{"a"}
	s2 := []string{"a", "b", "c", "d", "e"}
	s3 := []string{"a", "b", "c", "d", "e"}
	s4 := []string{"a", "b", "c"}
	s5 := s3[0:3]

	// arrays
	emptyArray := [0]string{}
	a1 := [3]string{"a"}
	a2 := [3]string{"a", "b", "c"}
	a3 := [3]string{"a", "b", "c"}

	tr := Tracing(t)
	mockT := &testing.T{}

	if ArrayEqual(mockT, s1, "a") || ArrayEqual(mockT, "a", s1) {
		tr.Errorf("expected ArrayEqual to fail on non-arrays")
	}

	if ArrayEqual(mockT, nilSlice, s1) {
		tr.Errorf("expected nil '%+v' not to equal '%+v'", nilSlice, s1)
	}
	if ArrayEqual(mockT, emptySlice, s1) {
		tr.Errorf("expected empty '%+v' not to equal '%+v'", emptySlice, s1)
	}
	if ArrayEqual(mockT, nilSlice, emptySlice) {
		tr.Errorf("expected nil '%+v' to not equal empty '%+v'", nilSlice, emptySlice)
	}
	if ArrayEqual(mockT, emptySlice, nilSlice) {
		tr.Errorf("expected empty '%+v' to not equal nil '%+v'", emptySlice, nilSlice)
	}
	if ArrayEqual(mockT, s1, s2) {
		tr.Errorf("expected '%+v' not to equal '%+v'", s1, s2)
	}
	if !ArrayEqual(mockT, s2, s3) {
		tr.Errorf("expected '%+v' to equal '%+v'", s2, s3)
	}
	if !ArrayEqual(mockT, s4, s5) {
		tr.Errorf("expected '%+v' to equal '%+v'", s4, s5)
	}

	if ArrayEqual(mockT, nilSlice, emptyArray) {
		tr.Errorf("expected nil '%+v' not to equal empty '%+v'", nilSlice, emptyArray)
	}

	if ArrayEqual(mockT, nilSlice, a1) {
		tr.Errorf("expected nil '%+v' not to equal '%+v'", nilSlice, a1)
	}

	if ArrayEqual(mockT, a1, a2) {
		tr.Errorf("expected '%+v' not to equal '%+v'", a1, a2)
	}

	if !ArrayEqual(mockT, a2, a3) {
		tr.Errorf("expected '%+v' to equal '%+v'", a2, a3)
	}

	if !ArrayEqual(mockT, s4, a2) {
		tr.Errorf("expected '%+v' to equal '%+v'", s4, a2)
	}

	if !ArrayEqual(mockT, s5, a2) {
		tr.Errorf("expected '%+v' to equal '%+v'", s5, a2)
	}

}

func TestMapEqual(t *testing.T) {
	var nilMap map[string]int
	emptyMap := map[string]int{}
	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"a": 1, "b": 2, "c": 3}
	m3 := map[string]int{"a": 1, "b": 2, "c": 3}
	m4 := map[string]int{"a": 99, "b": 2, "c": 3}
	m5 := map[string]int{"a": 1, "b": 2}

	tr := Tracing(t)
	mockT := &mockT{}

	if MapEqual(mockT, m1, "a") || MapEqual(mockT, "a", m1) {
		tr.Errorf("expected MapEqual to fail on non-arrays")
	}
	if MapEqual(mockT, nilMap, emptyMap) {
		tr.Errorf("expected nil '%+v' not to equal empty '%+v'", nilMap, emptyMap)
	}
	if MapEqual(mockT, emptyMap, nilMap) {
		tr.Errorf("expected empty '%+v' not to equal nil '%+v'", emptyMap, nilMap)
	}
	if MapEqual(mockT, m1, m2) {
		tr.Errorf("expected '%+v' not to equal '%+v'", m1, m2)
	}
	if !MapEqual(mockT, m2, m3) {
		tr.Errorf("expected '%+v' to equal '%+v'", m2, m3)
	}

	mockT.reset()
	if MapEqual(mockT, m3, m4) {
		tr.Errorf("expected '%+v' not to equal '%+v'", m3, m4)
	}
	mockT.checkErrorPrefix(
		tr,
		"maps not equal:\nkey `a`: got (int) 1, want (int) 99 in ",
	)

	mockT.reset()
	if MapEqual(mockT, m1, m5) {
		tr.Errorf("expected '%+v' not to equal '%+v'", m1, m5)
	}
	mockT.checkErrorPrefix(
		tr,
		"maps not equal:\nmissing key `b`: wanted value: (int) 2 in ",
	)

	mockT.reset()
	if MapEqual(mockT, m5, m1) {
		tr.Errorf("expected '%+v' not to equal '%+v'", m5, m1)
	}
	mockT.checkErrorPrefix(
		tr,
		"maps not equal:\nextra key `b`: unwanted value: (int) 2 in ",
	)
}

func TestNotDeepEqual(t *testing.T) {
	testCases := append(equalityTestCases, deepEqualityTestCases...)
	for _, test := range testCases {
		test.run(
			t,
			NotDeepEqual,
			func(test equalTestCase) bool {
				return test.kind == notEqual || test.kind == justEqual || test.kind == justJsonEqual
			})
	}
}

func TestSameInstance(t *testing.T) {
	for _, test := range sameInstanceTestCases {
		test.run(
			t,
			SameInstance,
			func(test equalTestCase) bool {
				return test.kind == justEqual || test.kind == equalAndDeepEqual
			})
	}
}

func TestNotSameInstance(t *testing.T) {
	for _, test := range sameInstanceTestCases {
		test.run(
			t,
			NotSameInstance,
			func(test equalTestCase) bool {
				return test.kind == notEqual || test.kind == justDeepEqual || test.kind == justJsonEqual
			})
	}
}

func TestEqualJson(t *testing.T) {
	testCases := append(append(equalityTestCasesJsonOk, deepEqualityTestCasesJsonOk...), justJsonEqualTestCases...)
	for _, test := range testCases {
		test.run(
			t,
			EqualJson,
			func(test equalTestCase) bool {
				return test.kind == justDeepEqual || test.kind == equalAndDeepEqual || test.kind == justEqual || test.kind == justJsonEqual
			})
	}
}

func TestNotEqualJson(t *testing.T) {
	testCases := append(append(equalityTestCasesJsonOk, deepEqualityTestCasesJsonOk...), justJsonEqualTestCases...)
	for _, test := range testCases {
		test.run(
			t,
			NotEqualJson,
			func(test equalTestCase) bool {
				return test.kind == notEqual
			})
	}
}

func TestMatchesRegex(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	if !MatchesRegex(mockT, "xyzpdq", "^xyz") {
		tr.Errorf("expected 'xyzpdq' to match '^xyz'")
	}
	if !MatchesRegex(mockT, "xyzpdq", "pdq$") {
		tr.Errorf("expected 'xyzpdq' to match 'pdq$'")
	}
	if !MatchesRegex(mockT, "xyzpdq", "zp") {
		tr.Errorf("expected 'xyzpdq' to match 'zp'")
	}
	if !MatchesRegex(mockT, "xyzpdq", "^xy.+dq$") {
		tr.Errorf("expected 'xyzpdq' to match '^xy.+dq$'")
	}

	if MatchesRegex(mockT, "xyzpdq", "a+") {
		tr.Errorf("expected 'xyzpdq' to not match 'a+'")
	}
}

func TestDoesNotMatchRegex(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	if DoesNotMatchRegex(mockT, "xyzpdq", "^xyz") {
		tr.Errorf("expected 'xyzpdq' to fail by matching '^xyz'")
	}

	if !DoesNotMatchRegex(mockT, "xyzpdq", "a+") {
		tr.Errorf("expected 'xyzpdq' to not match 'a+'")
	}
}

func TestErrorContains(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	err := fmt.Errorf("this error contains: magic!")

	if !ErrorContains(mockT, err, "magic") {
		tr.Errorf("expected '%s' to contain 'magic'", err.Error())
	}
	if ErrorContains(mockT, err, "special sauce") {
		tr.Errorf("expected '%s' not to contain 'special sauce'", err.Error())
	}
	if ErrorContains(mockT, nil, "anything") {
		tr.Errorf("expected nil error not to pass check")
	}
}

func TestErrorDoesNotContain(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	err := fmt.Errorf("this error contains: magic!")

	if ErrorDoesNotContain(mockT, err, "magic") {
		tr.Errorf("expected '%s' to contain 'magic', but it did", err.Error())
	}
	if !ErrorDoesNotContain(mockT, err, "special sauce") {
		tr.Errorf("expected '%s' not to contain 'special sauce'", err.Error())
	}
	if ErrorDoesNotContain(mockT, nil, "anything") {
		tr.Errorf("expected nil error not to pass check")
	}
}

func TestStringContains(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	str := "this string contains: magic!"

	if !StringContains(mockT, str, "magic") {
		tr.Errorf("expected '%s' to contain 'magic'", str)
	}
	if StringContains(mockT, str, "special sauce") {
		tr.Errorf("expected '%s' not to contain 'special sauce'", str)
	}
	if StringContains(mockT, "", "anything") {
		tr.Errorf("expected '' not to contain 'anything'")
	}
}

func TestStringDoesNotContain(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	str := "this error contains: magic!"

	if StringDoesNotContain(mockT, str, "magic") {
		tr.Errorf("expected '%s' to contain 'magic', but it did", str)
	}
	if !StringDoesNotContain(mockT, str, "special sauce") {
		tr.Errorf("expected '%s' not to contain 'special sauce'", str)
	}
	if !StringDoesNotContain(mockT, "", "anything") {
		tr.Errorf("expected '' not to contain 'anything'")
	}
}

func TestHasSameElements(t *testing.T) {
	tr := Tracing(t)
	mockT := &testing.T{}

	expectSame := func(a, b interface{}) {
		if !HasSameElements(mockT, a, b) {
			tr.Errorf("expected '%v' to have same elements as '%v'", a, b)
		}
	}

	expectDifferent := func(a, b interface{}) {
		if HasSameElements(mockT, a, b) {
			tr.Errorf("expected '%v' to not have same elements as '%v'", a, b)
		}
	}

	a1 := []int{1, 2, 3}
	a2 := []int{3, 2, 1}
	a3 := []int{1, 1, 1}
	a4 := []int{1, 2, 3, 4}
	a5 := []int{1, 1, 1, 2, 2, 2}
	a6 := []int{1, 2, 1, 2, 1, 2}
	a7 := []int{1, 1, 2, 2}

	expectSame(a1, a2)
	expectDifferent(a3, a1)
	expectDifferent(a1, a3)
	expectDifferent(a1, a4)
	expectDifferent(a4, a1)
	expectSame(a5, a6)
	expectDifferent(a5, a7)

	a8 := []complexStruct{cs1a, cs2b}
	a9 := []complexStruct{cs2b, cs1a}
	a10 := []complexStruct{cs1a, cs4}

	expectSame(a8, a9)
	expectDifferent(a8, a10)

	big_array := []int{1, 2, 3, 4, 5, 6, 5, 4, 3, 2, 1}
	s1 := big_array[0:5]
	s2 := big_array[6:]
	s3 := big_array[3:9]

	expectSame(s1, s2)
	expectDifferent(s1, s3)

	c1 := make(chan string, 10)
	c2 := make(chan string, 10)
	c3 := make(chan string, 10)

	for _, ch := range []string{"a", "b", "c"} {
		c1 <- ch
		c2 <- ch + ch
		c3 <- ch
	}
	close(c1)
	close(c2)
	// do not close c3

	expectSame(c1, []string{"a", "b", "c"})
	expectDifferent(c2, []string{"a", "b", "c"})
	expectSame(c3, []string{"a", "b", "c"})
}

func TestHasSameElementsInternals(t *testing.T) {
	mockT := &testing.T{}
	tr := Tracing(t)

	strType := reflect.TypeOf("x")

	intArray := []int{1, 2, 3}
	intChan := make(chan int, 1)

	intArrayType := reflect.TypeOf(intArray)
	intSliceType := reflect.TypeOf(intArray[0:1])
	intChanType := reflect.TypeOf(intChan)

	strArray := []string{"a", "b", "c"}
	strChan := make(chan string, 1)
	var strSendChan chan<- string
	strSendChan = strChan

	strArrayType := reflect.TypeOf(strArray)
	strSliceType := reflect.TypeOf(strArray[0:1])
	strChanType := reflect.TypeOf(strChan)
	strSendChanType := reflect.TypeOf(strSendChan)

	acceptableCases := [][]reflect.Type{
		{intArrayType, intArrayType},
		{intSliceType, intArrayType},
		{intArrayType, intSliceType},
		{intSliceType, intSliceType},
		{intChanType, intArrayType},
		{intChanType, intSliceType},
		{strArrayType, strArrayType},
		{strSliceType, strArrayType},
		{strArrayType, strSliceType},
		{strSliceType, strSliceType},
		{strChanType, strArrayType},
		{strChanType, strSliceType},
	}

	unacceptableCases := [][]reflect.Type{
		{strType, strArrayType},
		{strType, strSliceType},
		{strArrayType, intArrayType},
		{intArrayType, strArrayType},
		{strChanType, strChanType},
		{strArrayType, strType},
		{strSendChanType, strChanType},
		{strSendChanType, strArrayType},
	}

	for i, testcase := range acceptableCases {
		gotType := testcase[0]
		wantType := testcase[1]
		if !checkContainerTypes(mockT, gotType, wantType) {
			tr.Errorf(
				"expected '%v' and '%v' to be accepted, but was not (case %d)",
				gotType,
				wantType,
				i)
		}
	}

	for i, testcase := range unacceptableCases {
		gotType := testcase[0]
		wantType := testcase[1]
		if checkContainerTypes(mockT, gotType, wantType) {
			tr.Errorf(
				"expected '%v' and '%v' to be rejected, but was not (case %d)",
				gotType,
				wantType,
				i)
		}
	}
}

func TestPanic(t *testing.T) {
	tr := Tracing(t)

	ok := func() int { return 1 }
	panicky := func() string { panic("oh noes") }

	mt := &mockT{}
	if Panic(mt, ok) {
		tr.Errorf("expected Panic to return false")
	}

	expectedPrefix := "expected panic in "
	if len(mt.log) != 1 || mt.log[0].op != "Error" || !strings.HasPrefix(mt.log[0].args, expectedPrefix) {
		tr.Errorf("got %+v, want single Error op starting with %q", mt.log, expectedPrefix)
	}

	mt = &mockT{}
	if !Panic(mt, panicky) {
		tr.Errorf("expected Panic to return true")
	}
	if len(mt.log) != 0 {
		tr.Errorf("Expected no testing.T operations, got: %v", mt.log)
	}

	mt = &mockT{}
	if Panic(mt, "what is this even?") {
		tr.Errorf("expected Panic to return false")
	}
	if len(mt.log) != 1 ||
		mt.log[0].op != "Errorf" ||
		!strings.Contains(mt.log[0].args, "must be a function") {
		tr.Errorf(
			"got %+v, wanted single Errorf op containing 'must be a function'",
			mt.log,
		)
	}

	mt = &mockT{}
	if Panic(mt, func(i int) int { return i + 1 }) {
		tr.Errorf("expected Panic to return false")
	}
	if len(mt.log) != 1 ||
		mt.log[0].op != "Errorf" ||
		!strings.Contains(mt.log[0].args, "may not take arguments") {
		tr.Errorf(
			"got %+v, wanted single Errorf op containing 'not not take arguments'",
			mt.log,
		)
	}
}

func TestSameInstanceNonPointers(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	tr := Tracing(t)

	expectedPrefix := "cannot determine instance equality for non-pointer type:"

	rescued := func(a interface{}) {
		defer wg.Done()
		defer func() {
			if p := recover(); p != nil {
				if s, ok := p.(string); ok {
					if !strings.HasPrefix(s, expectedPrefix) {
						tr.Errorf("wrong panic message, got %q", s)
					}
				} else {
					tr.Errorf("expected panic string, got '%+v'", p)
				}
			} else {
				tr.Errorf("expected panic comparing '%+v' to itself", a)
			}
		}()

		sameInstance(a, a)
	}

	go rescued("a")
}

func TestEqualWithNonPrintableStings(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	a := "xyz"
	b := "xyz\x00"

	if Equal(mockT, a, b) {
		tr.Errorf("expected null-terminated inequality")
	}

	if len(mockT.log) != 1 {
		tr.Errorf("expected a single log entry, got: %+v", mockT.log)
	}

	log := mockT.log[0]
	expectedPrefix := "got (string) `xyz`, want (string) \"xyz\\x00\" in "
	if !strings.HasPrefix(log.args, expectedPrefix) {
		tr.Errorf("got %q, expected prefix %q", log.args, expectedPrefix)
	}
}

func TestGroupPassing(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	Group("name", mockT, func(g *G) {
		True(g, true)
		g.Group("sub-group", func(g *G) {
			False(g, false)
		})
	})

	if len(mockT.log) != 0 {
		tr.Errorf("Expected no testing.T operations, got: %v", mockT.log)
	}
}

func TestGroupFailing(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	Group("name", mockT, func(g *G) {
		True(g, false)
		g.Group("sub-group", func(g *G) {
			False(g, false)
		})
	})

	if len(mockT.log) != 1 || mockT.log[0].op != "Error" {
		tr.Errorf("got %+v, want single Error op", mockT.log)
	}

	expectedPrefix := "name: "
	if !strings.HasPrefix(mockT.log[0].args, expectedPrefix) {
		tr.Errorf("got '%s', want prefix '%s'", mockT.log[0].args, expectedPrefix)
	}
}

func TestNestedGroupFailing(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	Group("main-group", mockT, func(g *G) {
		True(g, true)
		g.Group("sub-group", func(g *G) {
			True(g, false)
		})
	})

	if len(mockT.log) != 1 || mockT.log[0].op != "Error" {
		tr.Errorf("got %+v, want single Error op", mockT.log)
	}

	expectedPrefix := "main-group sub-group: "
	if !strings.HasPrefix(mockT.log[0].args, expectedPrefix) {
		tr.Errorf("got '%s', want prefix '%s'", mockT.log[0].args, expectedPrefix)
	}
}

func TestGroupErrof(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	Group("main-group", mockT, func(g *G) {
		g.Errorf("failed %s", "here")
	})

	if len(mockT.log) != 1 || mockT.log[0].op != "Errorf" {
		tr.Errorf("got %+v, want single Errorf op", mockT.log)
	}

	expectedPrefix := "main-group: failed here in "
	if !strings.HasPrefix(mockT.log[0].args, expectedPrefix) {
		tr.Errorf("got '%s', want prefix '%s'", mockT.log[0].args, expectedPrefix)
	}
}

func TestGroupFatal(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	Group("main-group", mockT, func(g *G) {
		g.Fatal("boom")
		g.Fatalf("boom %s", "two")
	})

	if len(mockT.log) != 2 {
		tr.Errorf("expected 2 ops, got %d", len(mockT.log))
	}

	if mockT.log[0].op != "Fatal" {
		tr.Errorf("got %+v, want Fatal op", mockT.log[0])
	}
	if mockT.log[1].op != "Fatalf" {
		tr.Errorf("got %+v, want Fatalf op", mockT.log[1])
	}

	expectedPrefix1 := "main-group: boom in "
	if !strings.HasPrefix(mockT.log[0].args, expectedPrefix1) {
		tr.Errorf("got '%s', want prefix '%s'", mockT.log[0].args, expectedPrefix1)
	}

	expectedPrefix2 := "main-group: boom two in "
	if !strings.HasPrefix(mockT.log[1].args, expectedPrefix2) {
		tr.Errorf("got '%s', want prefix '%s'", mockT.log[1].args, expectedPrefix2)
	}
}

func TestUngrouped(t *testing.T) {
	tr := Tracing(t)
	mockT := &mockT{}

	True(mockT, false)

	if len(mockT.log) != 1 || mockT.log[0].op != "Error" {
		tr.Errorf("got %+v, want single Error op", mockT.log)
	}

	expectedPrefix := "got (bool) false, want (bool) true"
	if !strings.HasPrefix(mockT.log[0].args, expectedPrefix) {
		tr.Errorf("got '%s', want prefix '%s'", mockT.log[0].args, expectedPrefix)
	}
}
