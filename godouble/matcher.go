/*
 * Copyright 2020 grant@lastweekend.com.au
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package godouble

import (
	"fmt"
	"reflect"
	"strings"
)

// Matcher is used to match a method signature or one argument at a time
type Matcher interface {

	//Matches returns true if the arg (or args) matches this matcher
	Matches(args ...interface{}) bool
}

// MethodArgsMatcher is a Matcher that can validate usage against a reflect.Method
type MethodArgsMatcher interface {
	Matcher
	//ForMethod uses t to assert suitability of this matcher to match the method signature of m
	ForMethod(t T, m reflect.Method)
}

// A SingleArgMatcher is a Matcher that can validate usage against a reflect.Type
type SingleArgMatcher interface {
	Matcher

	//ForType uses t to assert suitability of this matcher to match a single argument of type ft
	ForType(t T, ft reflect.Type)
}

// A CombinationMatcher is a Matcher than can validated usage for both methods and types
type CombinationMatcher interface {
	Matcher
	ForMethod(t T, m reflect.Method)
	ForType(t T, ft reflect.Type)
}

func forMethod(t T, method reflect.Method, matcher Matcher) {
	if mam, isMAM := matcher.(MethodArgsMatcher); isMAM {
		mam.ForMethod(t, method)
	} else {
		t.Fatalf("Cannot use %v as MethodArgsMatcher", matcher)
	}
}

func forType(t T, ft reflect.Type, matcher Matcher) {
	if sam, isSAM := matcher.(SingleArgMatcher); isSAM {
		sam.ForType(t, ft)
	} else {
		t.Fatalf("Cannot use %v as SingleArgMatcher", matcher)
	}
}

func genericSingleArgumentMatcher(matcher interface{}) SingleArgMatcher {
	switch typedMatcher := matcher.(type) {
	case SingleArgMatcher:
		return typedMatcher
	case reflect.Type:
		return IsA(typedMatcher)
	default:
		if reflect.TypeOf(matcher).Kind() == reflect.Func {
			return Func(matcher)
		} else {
			return Eql(matcher)
		}
	}
}

func NewMatcherForMethod(t T, forMethod reflect.Method, matchers ...interface{}) (result MethodArgsMatcher) {
	forType := forMethod.Type
	if forType.NumIn() == 0 {
		t.Fatalf("Cannot build matcher for %v which takes no arguments", forMethod)
	}

	if len(matchers) == 0 {
		return All()
	}

	if reflect.TypeOf(matchers[0]).Kind() == reflect.Func {
		if len(matchers) > 1 {
			result = Func(matchers[0], matchers[1:]...)
		} else {
			result = Func(matchers[0])
		}

	} else if len(matchers) > 1 {
		matcherSlice := make([]Matcher, len(matchers))
		for i, m := range matchers {
			matcherSlice[i] = genericSingleArgumentMatcher(m)
		}
		return Args(matcherSlice...)

	} else if m, isMatcher := matchers[0].(MethodArgsMatcher); isMatcher {
		result = m
	} else {
		result = Args(genericSingleArgumentMatcher(matchers[0]))
	}

	result.ForMethod(t, forMethod)

	return
}

type funcMatcher struct {
	reflect.Value
	explanation string
}

func (f funcMatcher) String() string {
	return f.explanation
}

func (f funcMatcher) ForMethod(t T, m reflect.Method) {
	t.Helper()
	ft := f.Value.Type()
	if ft.Kind() != reflect.Func || ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.Bool {
		t.Fatalf("expected Func(...) bool, have %v", ft)
	}

	AssertMethodInputs(t, m, ft)
}

func (f funcMatcher) ForType(t T, in reflect.Type) {
	t.Helper()
	vt := f.Type()
	if f.Kind() != reflect.Func || vt.NumIn() != 1 || vt.NumOut() != 1 || vt.Out(0).Kind() != reflect.Bool {
		t.Fatalf("%v expected to be a function that accepts 1 argument and returns bool, got %v", f, vt)
	}
	if !in.AssignableTo(vt.In(0)) {
		t.Fatalf("Argument to %v expected to be assignable from %v, got %v", f, in, vt.In(0))
	}
}

func (f funcMatcher) Matches(args ...interface{}) bool {
	inArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		inArgs[i] = reflect.ValueOf(arg)
	}

	if f.Type().IsVariadic() {
		return f.CallSlice(inArgs)[0].Interface().(bool)
	}
	return f.Call(inArgs)[0].Interface().(bool)
}

//Func returns a matcher from the arbitrary function f
// Custom matcher methods will generally be a wrapper around Func
//
// When used as a method args matcher f(...) bool must have a compatible argument signature with the stubbed method
// When used as a single arg matcher f must be a func(x T) bool where T is assignable from the equivalent arg in the
// stubbed method
// Optionally include an explanation that will be formatted to string to describes what is being matched
func Func(f interface{}, explanation ...interface{}) CombinationMatcher {
	fv := reflect.ValueOf(f)

	var explainString string
	if len(explanation) == 0 {
		explainString = fmt.Sprintf("%T", f)
	} else {
		explainString = fmt.Sprint(explanation...)
	}

	return funcMatcher{fv, explainString}
}

type matcherList []Matcher

func (l matcherList) toString(prefix string, lRune rune, rRune rune) string {
	s := strings.Builder{}
	s.WriteString(prefix)
	if len(l) > 0 {
		s.WriteRune(lRune)
		for i, arg := range l {
			if i > 0 {
				s.WriteRune(',')
			}
			s.WriteString(fmt.Sprint(arg))
		}
		s.WriteRune(rRune)
	}
	return s.String()
}

func (l matcherList) ForMethod(t T, m reflect.Method) {
	t.Helper()
	for _, matcher := range l {
		forMethod(t, m, matcher)
	}
}

func (l matcherList) ForType(t T, ft reflect.Type) {
	t.Helper()
	for _, matcher := range l {
		forType(t, ft, matcher)
	}
}

type argumentsMatcher struct {
	matcherList matcherList
}

func (l *argumentsMatcher) Matches(args ...interface{}) bool {
	for i := 0; i < len(l.matcherList) && i < len(args); i++ {
		matcher, arg := l.matcherList[i], args[i]
		if !matcher.Matches(arg) {
			return false
		}
	}
	return true
}

func (l *argumentsMatcher) ForMethod(t T, m reflect.Method) {
	t.Helper()
	methodType := m.Type

	if methodType.IsVariadic() {
		if len(l.matcherList) > methodType.NumIn()-1 {
			//collapse m.NumIn - 1 ... len(args) - 1 into a Slice() matcher
			newMatchers := make([]Matcher, methodType.NumIn())
			copy(newMatchers, l.matcherList[:methodType.NumIn()-1])
			sliceMatchers := make([]Matcher, len(l.matcherList)-methodType.NumIn()+1)
			copy(sliceMatchers, l.matcherList[methodType.NumIn()-1:])
			newMatchers[methodType.NumIn()-1] = Slice(sliceMatchers...)
			l.matcherList = newMatchers
		}
	} else if len(l.matcherList) > methodType.NumIn() {
		t.Fatalf("%v requires not more than %d argument matchingArguments, have %d", m, methodType.NumIn(), len(l.matcherList))
	}

	for i, matcher := range l.matcherList {
		if sam, ok := matcher.(SingleArgMatcher); ok {
			sam.ForType(t, methodType.In(i))
		} else {
			t.Fatalf("Cannot validate %v as SingleArgMatcher for %v", matcher, methodType.In(i))
		}
	}
}

func (l *argumentsMatcher) String() string {
	return l.matcherList.toString("Args", '(', ')')
}

// Args builds a method arguments matcher from a list of single ArgumentMatchers
func Args(matchers ...Matcher) MethodArgsMatcher {
	return &argumentsMatcher{matchers}
}

type sliceMatcher struct {
	matcherList
}

//Slice returns a Matcher for a Slice type from a list of other SingleArgumentMatchers
//
//If all the matcherList match the argument in the corresponding position of the newSliceMatcher
func Slice(matchers ...Matcher) SingleArgMatcher {
	return &sliceMatcher{matchers}
}

func (sm *sliceMatcher) String() string {
	return sm.toString("Slice", '[', ']')
}

func (sm *sliceMatcher) Matches(args ...interface{}) bool {
	slice := args[0]
	v := reflect.ValueOf(slice)
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		if v.Len() < len(sm.matcherList) {
			return false
		}
		for i := 0; i < len(sm.matcherList); i++ {
			if !sm.matcherList[i].Matches(v.Index(i).Interface()) {
				return false
			}
		}
		//if we have fewer args, than the newSliceMatcher has members, assume the remaining matcherList are "All() (which always matches"
		return true

	default:
		return false
	}
}

func (sm *sliceMatcher) ForType(t T, in reflect.Type) {
	t.Helper()
	if in.Kind() != reflect.Slice && in.Kind() != reflect.Array {
		t.Fatalf("Slice() used to match non slice or array type %v", in)
	} else {
		sm.matcherList.ForType(t, in.Elem())
	}
}

// Eql matches a single argument v via reflect.DeepEqual
func Eql(v interface{}) SingleArgMatcher {
	return Func(func(arg interface{}) bool {
		return reflect.DeepEqual(arg, v)
	}, "Eql", "(", v, ")")
}

type nilMatcher struct{}

func (n nilMatcher) String() string {
	return "Nil"
}

func (n nilMatcher) Matches(args ...interface{}) bool {
	arg := args[0]
	if arg == nil {
		return true
	}

	v := reflect.ValueOf(arg)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}

func (n nilMatcher) ForType(t T, ft reflect.Type) {
	t.Helper()
	switch ft.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		//ok
	default:
		t.Fatalf("type %v cannot be nil", ft)
	}
}

var singletonNilMatcher = nilMatcher{}

// Nil matches a single argument of any nil-able type to be nil (or equivalent)
func Nil() SingleArgMatcher {
	return singletonNilMatcher
}

type lenMatcher struct {
	SingleArgMatcher
}

func (l lenMatcher) String() string {
	return fmt.Sprintf("Len(%v)", l.SingleArgMatcher)
}

func (l lenMatcher) Matches(args ...interface{}) bool {
	arg := args[0]
	v := reflect.ValueOf(arg)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return l.SingleArgMatcher.Matches(v.Len())
	default:
		return false
	}
}

func (l lenMatcher) ForType(t T, ft reflect.Type) {
	t.Helper()
	switch ft.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		l.SingleArgMatcher.ForType(t, reflect.TypeOf(0))
	default:
		t.Fatalf("cannot check length of type %v", ft)
	}
}

// Len matches a single argument that is type Array,Chan,Map,Slice,String type that have length matching v
//
// l may be anything that can match an int
// eg
//   Len(0)
//   Len(func(l int) bool { l <= 10})
func Len(v interface{}) SingleArgMatcher {
	return lenMatcher{genericSingleArgumentMatcher(v)}
}

//IsA matches a single argument if the supplied argument is AssignableTo or Implements the reflect.Type t
//
// if t is not already a reflect.Type it will be converted with reflect.TypeOf
func IsA(t interface{}) SingleArgMatcher {
	rt, isType := t.(reflect.Type)
	if !isType {
		rt = reflect.TypeOf(t)
	}
	return Func(func(x interface{}) bool {
		argT := reflect.TypeOf(x)
		switch argT.Kind() {
		case reflect.Interface:
			return argT.AssignableTo(rt) || argT.Implements(rt)
		default:
			return argT.AssignableTo(rt)
		}
	}, "IsA", "(", rt, ")")
}

type combinationMatcher struct {
	matcherList
	explain string
}

func (a combinationMatcher) String() string {
	return a.matcherList.toString(a.explain, '{', '}')
}

func newCombinationMatcher(matchers []Matcher, explain string) combinationMatcher {
	return combinationMatcher{matchers, explain}
}

type andMatcher struct {
	combinationMatcher
}

func (a andMatcher) Matches(args ...interface{}) bool {
	for _, m := range a.matcherList {
		if !m.Matches(args...) {
			return false
		}
	}
	return true
}

// All matches if all the matcherList match (returns true for no matchers)
func All(matchers ...Matcher) CombinationMatcher {
	return andMatcher{newCombinationMatcher(matchers, "All")}
}

// And matches if all the matcherList match
func And(matchers ...Matcher) CombinationMatcher {
	return All(matchers...)
}

type orMatcher struct {
	combinationMatcher
}

func (a orMatcher) Matches(arg ...interface{}) bool {
	for _, m := range a.matcherList {
		if m.Matches(arg...) {
			return true
		}
	}
	return false
}

// Any matches if any one of matcherList match (returns false for no matchers)
func Any(matchers ...Matcher) CombinationMatcher {
	return orMatcher{newCombinationMatcher(matchers, "Any")}
}

// Or matches if any one of matcherList match
func Or(matchers ...Matcher) CombinationMatcher {
	return Any(matchers...)
}

type notMatcher struct {
	Matcher
}

func (nm notMatcher) String() string {
	return fmt.Sprintf("Not(%v)", nm.Matcher)
}

func (nm notMatcher) Matches(arg ...interface{}) bool {
	return !nm.Matcher.Matches(arg...)
}

func (nm notMatcher) ForType(t T, ft reflect.Type) {
	t.Helper()
	forType(t, ft, nm.Matcher)
}

func (nm notMatcher) ForMethod(t T, m reflect.Method) {
	t.Helper()
	forMethod(t, m, nm.Matcher)
}

// Not negates matcher
func Not(matcher Matcher) CombinationMatcher {
	return notMatcher{matcher}
}
