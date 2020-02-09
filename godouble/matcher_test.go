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
	"regexp"
	"testing"
)

type tiface interface {
	test()
}

type tstring string

func (tstring) test() {
	panic("Unexpected call to test()")
}

var apiIface = reflect.TypeOf((*api)(nil)).Elem()

func TestMatcher(t *testing.T) {

	type test struct {
		name        string
		matcher     Matcher
		method      reflect.Method
		matching    []interface{}
		notMatching []interface{}
	}

	var apiF = func(name string) reflect.Method {
		m, ok := apiIface.MethodByName(name)
		if !ok {
			t.Fatalf("No method %s for %v", name, apiIface)
		}
		return m
	}

	ts := tstring("atest")

	regexF := func(x string) bool { return regexp.MustCompile("^t").MatchString(x) }
	tests := []test{
		{"callArg", Args(Eql("test")), apiF("call"), []interface{}{"test"}, []interface{}{""}},
		{"Func", Func(regexF), apiF("call"), []interface{}{"test"}, []interface{}{""}},
		{"Any", Any(Args(Eql("test")), Args(Eql("x"))), apiF("call"), []interface{}{"test"}, []interface{}{"xxx"}},
		{"Any", Any(Args(Func(regexF, "startswith 't'")), Args(Len(3))), apiF("call"), []interface{}{"yyy"}, []interface{}{"xxxx"}},
		{"All()", All(), apiF("call"), []interface{}{"ttt"}, nil},
		{"NoMatchers", NewMatcherForMethod(t, apiF("call")), apiF("call"), []interface{}{"ttt"}, nil},
		{"Any()", Any(), apiF("call"), nil, []interface{}{"ttt"}},
		{"All", All(Args(Func(regexF, "startswith 't'")), Args(Len(3))), apiF("call"), []interface{}{"ttt"}, []interface{}{"test"}},
		{"Not(Any(...))", Not(Any(Args(Func(regexF)), Args(Len(3)))), apiF("call"), []interface{}{"xxxx"}, []interface{}{"ttt"}},
		{"callNew", NewMatcherForMethod(t, apiF("call"), "test"), apiF("call"), []interface{}{"test"}, []interface{}{""}},
		{"callNewFunc", NewMatcherForMethod(t, apiF("call"), regexF), apiF("call"), []interface{}{"tight"}, []interface{}{""}},
		{"callNewFunc", NewMatcherForMethod(t, apiF("call"), regexF, "startswith 't'"), apiF("call"), []interface{}{"tight"}, []interface{}{""}},
		{"callNewType", NewMatcherForMethod(t, apiF("call"), IsA(ts)), apiF("call"), []interface{}{ts}, []interface{}{"plainstring"}},
		{"callNewIface", NewMatcherForMethod(t, apiF("call"), reflect.TypeOf((*tiface)(nil)).Elem()), apiF("call"), []interface{}{ts}, []interface{}{"plainstring"}},
		{"variadicArg", Args(Eql(10), Eql("test"), Eql("blah")), apiF("variadic"), []interface{}{10, []string{"test", "blah"}}, []interface{}{5, []string{"test"}}},
		{"pointersArg", Args(Eql(10), Eql("test")), apiF("pointers"), []interface{}{10, "test"}, []interface{}{5, "test"}},
		{"pointersNew", NewMatcherForMethod(t, apiF("pointers"), 10, "test"), apiF("pointers"), []interface{}{10, "test"}, []interface{}{5, "test"}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			vMatcher := test.matcher.(MethodArgsMatcher)
			vMatcher.ForMethod(t, test.method)
			if test.matching != nil && !test.matcher.Matches(test.matching...) {
				t.Errorf("Expected %v to match %v", test.matcher, test.matching)
			}
			if test.notMatching != nil && test.matcher.Matches(test.notMatching...) {
				t.Errorf("expected %v not to match %v", test.matcher, test.notMatching)
			}
		})
	}
}

func TestMethodArgsMatcher_FailsFatally(t *testing.T) {
	type test struct {
		name        string
		matcher     MethodArgsMatcher
		failMethod  reflect.Method
		expectedMsg string
	}

	var apiF = func(name string) reflect.Method {
		m, ok := apiIface.MethodByName(name)
		if !ok {
			t.Fatalf("No method %s for %v", name, apiIface)
		}
		return m
	}

	tests := []test{
		{"Any(Nil())", Any(Nil()), apiF("call"), "Nil as MethodArgsMatcher"},
		{"FuncTooManyArgs", Args(Eql("x"), Eql(0)), apiF("call"), "1.*have.*2"},
		{"FuncBadType", Func(func(i int) bool { return true }), apiF("call"), "string.*int"},
		{"ArgBadType", Args(Slice(Eql("xxx"))), apiF("call"), "slice.*string"},
		{"ArgBadType", Args(Args(Eql("x"))), apiF("call"), "SingleArgMatcher"},
		{"FuncNoReturn", Func(func(s string) {}), apiF("call"), "bool"},
		{"FuncMoreReturns", Func(func(s string) (bool, error) { return false, nil }), apiF("call"), "bool"},
		{"FuncReturnNotBool", Func(func(s string) error { return nil }), apiF("call"), "bool.*error"},
		{"FuncArgType", Args(Len(3)), apiF("test"), "length.*int"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			tDouble := NewTDouble(t, func(c *TestDouble) {
				//c.EnableTrace()
			})
			spy := tDouble.Fake("Fatalf", tDouble.FakeFatalf)
			defer func(spy FakeMethodCall) {
				recover()
				spy.Matching(printfMatcher(test.expectedMsg)).Expect(Once())
			}(spy)

			test.matcher.ForMethod(tDouble, test.failMethod)
		})
	}
}

func TestSingleArgMatchers(t *testing.T) {
	type test struct {
		name        string
		matcher     Matcher
		argType     reflect.Type
		matching    []interface{}
		notMatching []interface{}
		re          string
	}

	var emptySlice = make([]int, 0)
	var nilSlice []int
	nilSlice = nil
	intType := reflect.TypeOf(10)
	strType := reflect.TypeOf("")
	sliceIntType := reflect.TypeOf(emptySlice)
	sliceStrType := reflect.TypeOf([]string{})

	tests := []test{
		{"Eql(string)", Eql("x"), strType, []interface{}{"x"}, []interface{}{"y", ""}, "x"},
		{"Eql(int)", Eql(10), intType, []interface{}{10}, []interface{}{6, -1, 0}, "10"},
		{"NotEql(int)", Not(Eql(10)), intType, []interface{}{6, -1, 0}, []interface{}{10}, "Not.*10"},
		{"Nil([]int)", Nil(), sliceIntType, []interface{}{nilSlice}, []interface{}{emptySlice, []int{1}}, "Nil"},
		{"Slice([]int)", Slice(Eql(10), Eql(20)), sliceIntType, []interface{}{[]int{10, 20}, []int{10, 20, 3}}, []interface{}{[]int{10}, []int{1, 20}, emptySlice, nilSlice, "astring"}, `\[.*10.*20.*\]`},
		{"Len([]int)", Len(Eql(2)), sliceIntType, []interface{}{[]int{0, 0}}, []interface{}{emptySlice, []int{1}, []int{1, 2, 3}, 0}, "Len.*2"},
		{"Len(string)", Len(Eql(3)), sliceStrType, []interface{}{"one"}, []interface{}{"", "12"}, "Len.*3"},
		{"Len(Func(func >=))", Len(Func(func(l int) bool { return l >= 2 })), sliceIntType, []interface{}{"one", "xx"}, []interface{}{"x", ""}, "Len.*func.*int.*bool"},
		{"Len(func >=)", Len(func(l int) bool { return l >= 2 }), sliceIntType, []interface{}{"one", "xx"}, []interface{}{"x", ""}, "Len.*func.*int.*bool"},
		{"All()", All(), strType, []interface{}{"one", 10, true, emptySlice}, []interface{}{}, "All()"},
		{"Any()", Any(), strType, []interface{}{}, []interface{}{"one", 10, true, emptySlice}, "Any()"},
		{"All", All(All(), Eql("xxx"), Len(3)), strType, []interface{}{"xxx"}, []interface{}{"yyy"}, "All.*All.*xxx.*Len.*3"},
		{"Any", Any(Eql("xxx"), Len(2)), strType, []interface{}{"xxx", "ab"}, []interface{}{"yyy", ""}, "Any.*xxx.*Len.*2"},
		{"IsA", IsA(111), intType, []interface{}{33}, []interface{}{"yyyy"}, "int"},
		{"IsAType", IsA(reflect.TypeOf(10)), intType, []interface{}{33}, []interface{}{"yyyy"}, "int"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			matcher, matching, notMatching, argType := test.matcher, test.matching, test.notMatching, test.argType

			if !regexp.MustCompile(test.re).MatchString(fmt.Sprint(matcher)) {
				t.Errorf("expected '%v' to match '%s'", matcher, test.re)
			}

			vMatcher := matcher.(SingleArgMatcher)
			vMatcher.ForType(t, argType)

			for _, arg := range matching {
				if !matcher.Matches(arg) {
					t.Errorf("Expected %s to match %v", matcher, arg)
				}
			}

			for _, notArg := range notMatching {
				if matcher.Matches(notArg) {
					t.Errorf("Expectes %s to not match %v", matcher, notArg)
				}
			}
		})
	}
}
func TestSingleArgMatcher_FailsFatally(t *testing.T) {
	type test struct {
		name        string
		matcher     SingleArgMatcher
		failType    reflect.Type
		expectedMsg string
	}

	tests := []test{
		{"NonNilable", Nil(), reflect.TypeOf(0), "int.*nil"},
		{"NonSlice", Slice(Eql(10)), reflect.TypeOf(0), "slice.*int"},
		{"Any(Args)", Any(Args()), reflect.TypeOf(0), "SingleArgMatcher"},
		{"MultiArgFunc", Func(func(i int, s string) bool { return false }), reflect.TypeOf(0), "1 arg.*bool"},
		{"NonBoolFunc", Func(func(i int) {}), reflect.TypeOf(0), "1 arg.*bool"},
		{"BadArgType", Func(func(s string) bool { return false }), reflect.TypeOf(0), "string.*int"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			tDouble := NewTDouble(t, func(c *TestDouble) {
				c.EnableTrace()
			})
			spy := tDouble.Fake("Fatalf", tDouble.FakeFatalf)
			defer func(spy FakeMethodCall) {
				recover()
				spy.Matching(printfMatcher(test.expectedMsg)).Expect(Once())
			}(spy)

			test.matcher.ForType(tDouble, test.failType)
		})
	}
}
