// The assert package include various simple test assertions, and a mechanism
// to group assertions together.
package assert

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/turbinelabs/test/check"
	"github.com/turbinelabs/test/stack"
)

const (
	goPath      = "/usr/local/"
	tbnHomePath = "TBN_FULL_HOME"
)

// A TracingTB embeds a testing.TB, overriding the Errorf and Fatalf methods to
// append stack traces.
type TracingTB struct {
	testing.TB
}

// Tracing wraps a testing.T or testing.TB so that stack traces are appended to
// all Errorf and Fatalf calls. If a TracingTB or G is supplied, it is returned
// unmodified
func Tracing(t testing.TB) testing.TB {
	switch obj := t.(type) {
	case *G:
		return obj
	case *TracingTB:
		return obj
	default:
		return &TracingTB{t}
	}
}

func stackTrace() string {
	tbnPath := os.Getenv(tbnHomePath) + "/"

	trace := stack.New()
	if tbnPath != "/" {
		trace.TrimPaths(tbnPath, goPath)
	}

	trace.PopFrames("test/")

	return "\n" + trace.Format(true)
}

func (tr *TracingTB) Errorf(format string, args ...interface{}) {
	tr.TB.Errorf(format+" in %s", append(args, stackTrace())...)
}

func (tr *TracingTB) Error(args ...interface{}) {
	args = append(args, "in", stackTrace())
	tr.TB.Error(args...)
}

func (tr *TracingTB) Fatalf(format string, args ...interface{}) {
	tr.TB.Fatalf(format+" in %s", append(args, stackTrace())...)
}

func (tr *TracingTB) Fatal(args ...interface{}) {
	args = append(args, "in", stackTrace())
	tr.TB.Fatal(args...)
}

// G represents a (possibly nested) group of assertions. The Name field is
// used as a prefix to the error message generated by each assertion. G embeds
// a testing.TB, which may be a testing.T or a TracingTB (which itself embeds a
// testing.TB)
type G struct {
	testing.TB
	Name string
}

// Creates a group of assertions with a common error message prefix. The prefix
// holds only for assertion methods invoked with the G instance passed to the
// given function:
//
//     func TestThing(t *testing.T) {
//         Group("group-name", t, func(g *assert.G) {
//                 assert.True(g, true)
//         })
//     }
//
// Note that testing.TB is an interface implemented by testing.T.
func Group(name string, t testing.TB, f func(*G)) {
	group := &G{Tracing(t), name}
	f(group)
}

// Creates a nested group of assertions with a common error message prefix.
// The parent group's Name and the given name are joined with a space
// separator to form the prefix for the new group. The prefix holds only for
// assertion methods invoked on the G instance passed to this given function:
//
//     func TestThing(t *testing.T) {
//         Group("group-name", t, func(g *assert.G) {
//                 g.Group("nested-group-name", func(ng *assert.G) {
//                         assert.True(ng, true)
//                 })
//         })
//     }
func (grp *G) Group(name string, f func(*G)) {
	nestedName := fmt.Sprintf("%s %s", grp.Name, name)
	nestedGrp := &G{Tracing(grp.TB), nestedName}
	f(nestedGrp)
}

func (grp *G) Errorf(format string, args ...interface{}) {
	if len(grp.Name) > 0 {
		prefix := fmt.Sprintf("%s: ", grp.Name)
		format = prefix + format
	}

	grp.TB.Errorf(format, args...)
}

func (grp *G) Error(args ...interface{}) {
	if len(grp.Name) > 0 {
		newArgs := make([]interface{}, 1, len(args)+1)
		newArgs[0] = fmt.Sprintf("%s:", grp.Name)
		args = append(newArgs, args...)
	}

	grp.TB.Error(args...)
}

func (grp *G) Fatalf(format string, args ...interface{}) {
	if len(grp.Name) > 0 {
		prefix := fmt.Sprintf("%s: ", grp.Name)
		format = prefix + format
	}

	grp.TB.Fatalf(format, args...)
}

func (grp *G) Fatal(args ...interface{}) {
	if len(grp.Name) > 0 {
		newArgs := make([]interface{}, 1, len(args)+1)
		newArgs[0] = fmt.Sprintf("%s:", grp.Name)
		args = append(newArgs, args...)
	}

	grp.TB.Fatal(args...)
}

//
// assert methods
//

// Nil asserts the nilness of got.
func Nil(t testing.TB, got interface{}) bool {
	if !check.IsNil(got) {
		Tracing(t).Errorf("got (%T) %s, want <nil>", got, stringify(got))
		return false
	}
	return true
}

// NonNil asserts the non-nilness of got.
func NonNil(t testing.TB, got interface{}) bool {
	if check.IsNil(got) {
		Tracing(t).Errorf("got (%T) %s, want <non-nil>", got, stringify(got))
		return false
	}
	return true
}

func mkErrorMsg(got, want interface{}) string {
	return mkErrorMsgWithExp(got, want, "want")
}

func mkErrorMsgWithExp(got, want interface{}, expectation string) string {
	return fmt.Sprintf(
		"got (%T) %s, %s (%T) %s",
		got,
		stringify(got),
		expectation,
		want,
		stringify(want),
	)
}

func stringify(i interface{}) string {
	var s string
	switch t := i.(type) {
	case string:
		s = t
	case *string:
		if t == nil {
			return "<nil>"
		}
		s = *t
	default:
		return fmt.Sprintf("%+v", i)
	}

	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	} else {
		return strconv.Quote(s)
	}
}

// Equal asserts that got == want, and will panic for types that can't
// be compared with ==.
func Equal(t testing.TB, got, want interface{}) bool {
	if got != want {
		Tracing(t).Error(mkErrorMsg(got, want))
		return false
	}
	return true
}

// Equal asserts that got != want, and will panic for types that can't
// be compared with !=.
func NotEqual(t testing.TB, got, want interface{}) bool {
	if got == want {
		Tracing(t).Error(mkErrorMsgWithExp(got, want, "want !="))
		return false
	}
	return true
}

func isArrayLike(i interface{}) bool {
	t := reflect.TypeOf(i)
	if t == nil {
		return false
	}
	kind := t.Kind()
	return kind == reflect.Array || kind == reflect.Slice
}

// panics if a is not an array
func arrayValues(a interface{}) []reflect.Value {
	aValue := reflect.ValueOf(a)
	if aValue.Kind() != reflect.Array && aValue.IsNil() {
		return nil
	}
	valueArray := make([]reflect.Value, aValue.Len())
	for i := range valueArray {
		valueArray[i] = aValue.Index(i)
	}
	return valueArray
}

func ArrayEqual(t testing.TB, got, want interface{}) bool {
	if !isArrayLike(got) || !isArrayLike(want) {
		Tracing(t).Error(mkErrorMsg(got, want))
		return false
	}

	gotValues := arrayValues(got)
	wantValues := arrayValues(want)

	if gotValues == nil && wantValues != nil {
		Tracing(t).Errorf("got (%T) nil, want (%T) %s", got, want, stringify(want))
		return false
	} else if wantValues == nil && gotValues != nil {
		Tracing(t).Errorf("got (%T) %s, want (%T) nil", got, stringify(got), want)
		return false
	}

	gotLen := len(gotValues)
	wantLen := len(wantValues)

	errors := []string{}
	for i := 0; i < gotLen || i < wantLen; i++ {
		var gotIface, wantIface interface{}
		gotValid := i < gotLen
		if gotValid {
			gotIface = gotValues[i].Interface()
		}

		wantValid := i < wantLen
		if wantValid {
			wantIface = wantValues[i].Interface()
		}

		var err string
		if gotValid && wantValid {
			if !reflect.DeepEqual(gotIface, wantIface) {
				err = fmt.Sprintf(
					"index %d: %s",
					i,
					mkErrorMsg(gotIface, wantIface),
				)
			}
		} else if gotValid {
			err = fmt.Sprintf(
				"index %d: got extra value: (%T) %s",
				i,
				gotIface,
				stringify(gotIface),
			)
		} else if wantValid {
			err = fmt.Sprintf(
				"index %d: missing wanted value: (%T) %s",
				i,
				wantIface,
				stringify(wantIface),
			)
		}

		if err != "" {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		Tracing(t).Errorf("arrays not equal:\n%s", strings.Join(errors, "\n"))
		return false
	}
	return true
}

func isMap(i interface{}) bool {
	t := reflect.TypeOf(i)
	return t != nil && t.Kind() == reflect.Map
}

func MapEqual(t testing.TB, got, want interface{}) bool {
	if !isMap(got) || !isMap(want) {
		Tracing(t).Error(mkErrorMsg(got, want))
		return false
	}

	wantValue := reflect.ValueOf(want)
	wantKeys := wantValue.MapKeys()

	gotValue := reflect.ValueOf(got)
	gotKeys := gotValue.MapKeys()

	if gotValue.IsNil() && !wantValue.IsNil() {
		Tracing(t).Errorf("got (%T) nil, want (%T) %s", got, want, stringify(want))
		return false
	} else if wantValue.IsNil() && !gotValue.IsNil() {
		Tracing(t).Errorf("got (%T) %s, want (%T) nil", got, stringify(got), want)
		return false
	}

	errors := []string{}
	for _, wantKey := range wantKeys {
		wantIface := wantValue.MapIndex(wantKey).Interface()

		var err string
		gotMapValue := gotValue.MapIndex(wantKey)
		if gotMapValue.IsValid() {
			gotIface := gotMapValue.Interface()
			if !reflect.DeepEqual(gotIface, wantIface) {
				err = fmt.Sprintf(
					"key %s: %s",
					stringify(wantKey.Interface()),
					mkErrorMsg(gotIface, wantIface),
				)
			}
		} else {
			err = fmt.Sprintf(
				"missing key %s: wanted value: (%T) %s",
				stringify(wantKey.Interface()),
				wantIface,
				stringify(wantIface),
			)
		}
		if err != "" {
			errors = append(errors, err)
		}
	}

	for _, gotKey := range gotKeys {
		wantMapValue := wantValue.MapIndex(gotKey)
		if !wantMapValue.IsValid() {
			gotIface := gotValue.MapIndex(gotKey).Interface()
			err := fmt.Sprintf(
				"extra key %s: unwanted value: (%T) %s",
				stringify(gotKey.Interface()),
				gotIface,
				stringify(gotIface),
			)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		Tracing(t).Errorf("maps not equal:\n%s", strings.Join(errors, "\n"))
		return false
	}

	return true
}

// DeepEqual asserts reflect.DeepEqual(got, want).
func DeepEqual(t testing.TB, got, want interface{}) bool {
	if isArrayLike(got) && isArrayLike(want) {
		return ArrayEqual(t, got, want)
	} else if isMap(got) && isMap(want) {
		return MapEqual(t, got, want)
	} else if !reflect.DeepEqual(got, want) {
		Tracing(t).Error(mkErrorMsg(got, want))
		return false
	}
	return true
}

// NotDeepEqual asserts !reflect.DeepEqual(got, want).
func NotDeepEqual(t testing.TB, got, want interface{}) bool {
	if reflect.DeepEqual(got, want) {
		Tracing(t).Error(mkErrorMsgWithExp(got, want, "want !="))
		return false
	}
	return true
}

func sameInstance(got, want interface{}) bool {
	gotType := reflect.TypeOf(got)
	if gotType != reflect.TypeOf(want) {
		return false
	}
	switch gotType.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Ptr,
		reflect.Slice:

		gotVal := reflect.ValueOf(got)
		wantVal := reflect.ValueOf(want)
		if gotVal.Pointer() != wantVal.Pointer() {
			return false
		}
		// slices of different lengths can still share a pointer
		if gotType.Kind() == reflect.Slice && gotVal.Len() != wantVal.Len() {
			return false
		}
		return true

	default:
		panic(fmt.Sprintf(
			"cannot determine instance equality for non-pointer type: %T",
			got,
		))
	}
}

// SameInstance asserts that got and want are the same instance of a
// pointer type, or are the same value of a literal type.
func SameInstance(t testing.TB, got, want interface{}) bool {
	if !sameInstance(got, want) {
		Tracing(t).Error(mkErrorMsgWithExp(got, want, "want same instance as"))
		return false
	}
	return true
}

// NotSameInstance asserts that got and want are not the same instance of a
// pointer type, and are not the same value of a literal type.
func NotSameInstance(t testing.TB, got, want interface{}) bool {
	if sameInstance(got, want) {
		Tracing(t).Error(mkErrorMsgWithExp(got, want, "want not same instance as"))
		return false
	}
	return true
}

func encodeJson(t testing.TB, got, want interface{}) (string, string) {
	gotJson, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("could not marshal json: %#v", err)
	}
	wantJson, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("could not marshal json: %#v", err)
	}
	return string(gotJson), string(wantJson)
}

// EqualJson asserts that got and want encode to the same JSON value.
func EqualJson(t testing.TB, got, want interface{}) bool {
	tr := Tracing(t)
	gotJson, wantJson := encodeJson(tr, got, want)
	return Equal(tr, gotJson, wantJson)
}

// NotEqualJson asserts that got and want do not encode to the same JSON value.
func NotEqualJson(t testing.TB, got, want interface{}) bool {
	tr := Tracing(t)
	gotJson, wantJson := encodeJson(tr, got, want)
	return NotEqual(tr, gotJson, wantJson)
}

func matchRegex(t testing.TB, got, wantRegex string) bool {
	matched, err := regexp.MatchString(wantRegex, got)
	if err != nil {
		t.Fatalf("invalid regular expression `%s`: %#v", wantRegex, err)
	}

	return matched
}

// MatchesRegex asserts that got matches the provided regular expression.
func MatchesRegex(t testing.TB, got, wantRegex string) bool {
	tr := Tracing(t)
	if !matchRegex(tr, got, wantRegex) {
		tr.Errorf("got %q, did not match `%s`", got, wantRegex)
		return false
	}

	return true
}

// DoesNotMatchRegex asserts that got does not match the provided regular expression.
func DoesNotMatchRegex(t testing.TB, got, wantRegex string) bool {
	tr := Tracing(t)
	if matchRegex(tr, got, wantRegex) {
		tr.Errorf("got %q, matched `%s`", got, wantRegex)
		return false
	}

	return true
}

// True asserts that value is true.
func True(t testing.TB, value bool) bool {
	return Equal(Tracing(t), value, true)
}

// True asserts that value is false.
func False(t testing.TB, value bool) bool {
	return Equal(Tracing(t), value, false)
}

// Failed logs msg and aborts the current test.
func Failed(t testing.TB, msg string) {
	Tracing(t).Fatalf("Failed: %s", msg)
}

// ErrorContains asserts that got contains want.
func ErrorContains(t testing.TB, got error, want string) bool {
	tr := Tracing(t)
	if got == nil {
		tr.Errorf("got nil error, wanted message containing %s", stringify(want))
		return false
	} else if !strings.Contains(got.Error(), want) {
		tr.Errorf(
			"got error %s, wanted message containing %s",
			stringify(got.Error()),
			stringify(want),
		)
		return false
	}

	return true
}

// ErrorDoesNotContain asserts that got does not contain want.
func ErrorDoesNotContain(t testing.TB, got error, want string) bool {
	tr := Tracing(t)
	if got == nil {
		tr.Errorf("got nil error, wanted message not containing %s", stringify(want))
		return false
	} else if strings.Contains(got.Error(), want) {
		tr.Errorf(
			"got error %s, wanted message not containing %s",
			stringify(got.Error()),
			stringify(want),
		)
		return false
	}

	return true
}

// StringContains asserts that got contains want.
func StringContains(t testing.TB, got, want string) bool {
	tr := Tracing(t)
	if !strings.Contains(got, want) {
		tr.Errorf(
			"got %s, wanted message containing %s",
			stringify(got),
			stringify(want),
		)
		return false
	}

	return true
}

// StringDoesNotContain asserts that got does not contain want.
func StringDoesNotContain(t testing.TB, got, want string) bool {
	tr := Tracing(t)
	if strings.Contains(got, want) {
		tr.Errorf(
			"got %s, wanted message not containing %s",
			stringify(got),
			stringify(want),
		)
		return false
	}

	return true
}

func checkContainerTypes(t testing.TB, gotType, wantType reflect.Type) bool {
	gotKind := gotType.Kind()
	wantKind := wantType.Kind()

	switch gotKind {
	case reflect.Array, reflect.Slice:
		// ok

	case reflect.Chan:
		if gotType.ChanDir()&reflect.RecvDir == 0 {
			t.Errorf("got type '%v', a non-receiving channel", gotType)
			return false
		}

	default:
		t.Errorf("got type '%v', can only compare arrays, slices, or channels", gotType)
		return false
	}

	if wantKind != reflect.Array && wantKind != reflect.Slice {
		// We only compare with Array/Slices
		t.Errorf(
			"got type '%v', want type must be an array or slice of %s, not '%v'",
			gotType,
			gotType.Elem(),
			wantType)
		return false
	}

	// The Array/Slice/Chan element types must match
	if gotType.Elem() != wantType.Elem() {
		t.Errorf(
			"got type '%v', wanted type '%v': contains types do not match",
			gotType,
			wantType)
		return false
	}

	return true
}

func assertSameArray(gotValue, wantValue []reflect.Value) string {
	gotLen := len(gotValue)
	wantLen := len(wantValue)

	unusedGotIndicies := make([]int, gotLen)
	for i := 0; i < gotLen; i++ {
		unusedGotIndicies[i] = i
	}

	unusedWantIndicies := make([]int, wantLen)
	for i := 0; i < wantLen; i++ {
		unusedWantIndicies[i] = i
	}

	for gotIndex, v := range gotValue {
		for _, wantIndex := range unusedWantIndicies {
			if wantIndex != -1 {
				w := wantValue[wantIndex]
				if reflect.DeepEqual(v.Interface(), w.Interface()) {
					unusedWantIndicies[wantIndex] = -1
					unusedGotIndicies[gotIndex] = -1
					break
				}
			}
		}
	}

	extra := []interface{}{}
	for _, gotIndex := range unusedGotIndicies {
		if gotIndex != -1 {
			extra = append(extra, gotValue[gotIndex].Interface())
		}
	}

	missing := []interface{}{}
	for _, wantIndex := range unusedWantIndicies {
		if wantIndex != -1 {
			missing = append(missing, wantValue[wantIndex].Interface())
		}
	}

	if gotLen != wantLen || len(extra) > 0 || len(missing) > 0 {
		missingStr := ""
		if len(missing) > 0 {
			missingStr = fmt.Sprintf("; missing elements: %s", stringify(missing))
		}

		extraStr := ""
		if len(extra) > 0 {
			extraStr = fmt.Sprintf("; extra elements: %s", stringify(extra))
		}

		gotValueStr := []string{}
		for _, gv := range gotValue {
			gotValueStr = append(
				gotValueStr,
				fmt.Sprintf("(%s) %s", gv.Type().Name(), stringify(gv)),
			)
		}
		wantValueStr := []string{}
		for _, wv := range wantValue {
			wantValueStr = append(
				wantValueStr,
				fmt.Sprintf("(%s) %s", wv.Type().Name(), stringify(wv)),
			)
		}

		return fmt.Sprintf(
			"got [%s] (len %d), wanted [%s] (len %d)%s%s",
			strings.Join(gotValueStr, ", "),
			gotLen,
			strings.Join(wantValueStr, ", "),
			wantLen,
			missingStr,
			extraStr)
	}

	return ""
}

// Compares two container-like values. The got parameter may be an
// array, slice, or channel. The want parameter must be an array or
// slice whose element type is the same as that of got. If got is
// a channel, all available values are consumed (until the channel
// either blocks or indicates it was closed). The got and want
// values are then compared without respect to order.
func HasSameElements(t testing.TB, got, want interface{}) bool {
	tr := Tracing(t)
	gotType := reflect.TypeOf(got)
	wantType := reflect.TypeOf(want)
	if !checkContainerTypes(tr, gotType, wantType) {
		return false
	}

	gotValue := reflect.ValueOf(got)

	wantValueArray := arrayValues(want)

	var msg string
	switch gotType.Kind() {
	case reflect.Array, reflect.Slice:
		gotValueArray := arrayValues(got)
		msg = assertSameArray(gotValueArray, wantValueArray)

	case reflect.Chan:
		gotValueArray := []reflect.Value{}
		for {
			v, ok := gotValue.TryRecv()
			if !ok {
				// blocked or closed
				break
			}
			gotValueArray = append(gotValueArray, v)
		}
		msg = assertSameArray(gotValueArray, wantValueArray)

	default:
		msg = fmt.Sprintf(
			"internal error: unexpected kind %v",
			gotType.Kind())
	}

	if msg != "" {
		tr.Errorf(msg)
		return false
	}

	return true
}

// v must be a zero-arg function
func checkPanic(v reflect.Value) (i interface{}) {
	defer func() {
		if x := recover(); x != nil {
			i = x
		}
	}()

	v.Call(nil)
	return
}

// Panic asserts that the given function panics. The f parameter must
// be a function that takes no arguments. It may, however, return any
// number of arguments.
func Panic(t testing.TB, f interface{}) bool {
	fType := reflect.TypeOf(f)
	if fType.Kind() != reflect.Func {
		Tracing(t).Errorf("parameter to Panic must be a function: %+v", f)
		return false
	}
	if fType.NumIn() != 0 {
		Tracing(t).Errorf("function passed to Panic may not take arguments: %+v", f)
		return false
	}

	if checkPanic(reflect.ValueOf(f)) == nil {
		Tracing(t).Error("expected panic")
		return false
	}
	return true
}
