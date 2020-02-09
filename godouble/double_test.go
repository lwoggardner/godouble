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
	"regexp"
	"strings"
	"testing"
)

type api interface {
	call(in string) int
	other() int
	empty()
	test(i int, s string) (int, error)
	variadic(i int, slist ...string)
	pointers(*int, *string)
}

type apiDouble struct {
	api
	*TestDouble
}

func (a *apiDouble) call(in string) int {
	a.TestDouble.T().Helper()
	return a.Invoke("call", in)[0].(int)
}

func (a *apiDouble) other() int {
	a.TestDouble.T().Helper()
	return a.Invoke("other")[0].(int)
}

func (a *apiDouble) empty() {
	a.TestDouble.T().Helper()
	a.Invoke("empty")
}

func (a *apiDouble) test(i int, s string) (r int, e error) {
	a.TestDouble.T().Helper()
	returns := a.Invoke("test", i, s)
	r, _ = returns[0].(int)
	e, _ = returns[1].(error)
	return
}

func (a *apiDouble) variadic(i int, s ...string) {
	a.TestDouble.T().Helper()
	a.Invoke("test", i, s)
}

func (a *apiDouble) pointers(i *int, s *string) {
	a.TestDouble.T().Helper()
	a.Invoke("test", i, s)
}

func newApiDouble(t T, configs ...func(c *TestDouble)) *apiDouble {
	return &apiDouble{TestDouble: NewDouble(t, (*api)(nil), configs...)}
}

func TestNewDouble_FailsImmediatelyIfNotAnInterface(t *testing.T) {
	tDouble := NewTDouble(t)

	spy := tDouble.Fake("Fatalf", tDouble.FakeFatalf)
	defer func(spy FakeMethodCall) {
		recover()
		spy.Matching(printfMatcher(`pointer to nil interface`)).Expect(Once())
	}(spy)
	NewDouble(tDouble, "string not interface")
	t.Errorf("Expect unreachable")
}

func TestTestDouble_Stub_FailsFatallyForBadInputs(t *testing.T) {
	type badInputs struct {
		name        string
		bad         func(d *apiDouble)
		expectedMsg string
	}

	tests := []badInputs{
		{"InvalidMethod", func(d *apiDouble) { d.Stub("notamethod") }, "notamethod"},
		{"InvalidReturns", func(d *apiDouble) { d.Stub("other").Returning("notanint") }, "string"},
		{"InvalidMatcher", func(d *apiDouble) { d.Stub("other").Matching(Func(func(i int) bool { return true })) }, "int"},
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

			d := newApiDouble(tDouble)
			test.bad(d)
			t.Errorf("Expect unreachable")

		})
	}
}

func TestTestDouble_Stub(t *testing.T) {
	d1 := newApiDouble(t)

	s1 := d1.Stub("call").Matching("second").Returning(1)
	assertMatch(t, s1, "call.*matching.*second")
	s2 := d1.Stub("call").Returning(99)
	assertMatch(t, s2, "call")
	assertNotMatch(t, s2, "matching")

	if i := d1.call("first"); i != 99 {
		t.Errorf("Expected first d1.call to return 99, got %d", i)
	}
	if i := d1.call("second"); i != 1 {
		t.Errorf("Expected second d1.call to return 1, got %d", i)
	}

}

func TestInvoke_SkipsNonMatchingMock(t *testing.T) {
	d1 := newApiDouble(t)
	defer d1.Verify()

	d1.Mock("call").Matching("second").Returning(1).Expect(Once())
	d1.Stub("call").Returning(99)

	if i := d1.call("first"); i != 99 {
		t.Errorf("Expected first d1.call to return 99, got %d", i)
	}
	if i := d1.call("second"); i != 1 {
		t.Errorf("Expected second d1.call to return 1, got %d", i)
	}
	if i := d1.call("third"); i != 99 {
		t.Errorf("Expected third d1.call to return 99, got %d", i)
	}
}

func TestInvoke_SkipsCompleteMock(t *testing.T) {
	d1 := newApiDouble(t)
	defer d1.Verify()

	d1.Mock("call").Returning(1).Expect(Twice())
	d1.Stub("call").Returning(99)

	if i := d1.call("first"); i != 1 {
		t.Errorf("Expected first d1.call to return 1, got %d", i)
	}
	if i := d1.call("second"); i != 1 {
		t.Errorf("Expected second d1.call to return 1, got %d", i)
	}
	if i := d1.call("third"); i != 99 {
		t.Errorf("Expected third d1.call to return 99, since the first mocks expectations are complete. got %d", i)
	}
}

func TestRunsMocksInSequence(t *testing.T) {
	d1 := newApiDouble(t)
	d2 := newApiDouble(t)

	defer Verify(d1, d2)

	m1 := d1.Mock("call").Returning(1).Expect(Once())
	m2 := d2.Mock("test").Returning(2, nil).Expect(Once())
	d1.Mock("call").Returning(3).Expect(Once()).After(m2)
	d1.Mock("call").Returning(99)

	ExpectInOrder(m1, m2)

	if i := d1.call("first"); i != 1 {
		t.Errorf("Expected first d1.call to return 1, got %d", i)
	}
	if i := d1.call("second"); i != 99 {
		t.Errorf("Expected second d1.call to return 99, since d2.other not called yet. got %d", i)
	}
	if r, _ := d2.test(0, ""); r != 2 {
		t.Error("Expected f2.other to return 2, as it has been called after d1.call")
	}
	if i := d1.call("third"); i != 3 {
		t.Errorf("Expected third d1.call to return 2, since d2.other has now been called, got %d", i)
	}
}

func TestTestDouble_VerifyErrorsForMocksWhoseExpectationsHaveNotBeenMet(t *testing.T) {
	doubleT := NewTDouble(t)
	spy := doubleT.Spy("Errorf") //use a spy because we're testing mock verify!

	d1 := newApiDouble(doubleT)
	d1.Mock("other").Expect(Once())

	d1.Verify()

	spy.Matching(printfMatcher("other")).Expect(Once())
}

func assertMatch(t *testing.T, s interface{}, re string) {
	t.Helper()
	toMatch := fmt.Sprint(s)
	if matched, err := regexp.MatchString(re, toMatch); err != nil {
		t.Errorf("error %s trying to match /%s/ to %s", err.Error(), re, toMatch)
	} else if !matched {
		t.Errorf("expected %s to match /%s/", toMatch, re)
	}
}

func assertNotMatch(t *testing.T, s interface{}, re string) {
	t.Helper()
	toMatch := fmt.Sprint(s)
	if matched, err := regexp.MatchString(re, toMatch); err != nil {
		t.Errorf("error %s trying to not match /%s/ to %s", err.Error(), re, toMatch)
	} else if matched {
		t.Errorf("expected %s not to match /%s/", toMatch, re)
	}
}

func TestTestDouble_Spy(t *testing.T) {
	doubleT := NewTDouble(t, func(c *TestDouble) {
		//c.EnableTrace()
	})
	defer doubleT.Verify()
	doubleT.Mock("Errorf").Matching(printfMatcher(`hello`)).Expect(Once())
	doubleT.Fake("Fatalf", doubleT.FakeFatalf)
	doubleT.Stub("Helper")
	doubleT.Stub("Logf")

	d1 := newApiDouble(doubleT)

	rc := NewReturnChannel(3)
	rc.Send(1)
	rc.Send(2)
	rc.Send(3)
	d1.Spy("call").Returning(rc)

	if i := d1.call("first"); i != 1 {
		t.Errorf("Expected spy to be invoked returning 1, got %d", i)
	}
	if i := d1.call("second"); i != 2 {
		t.Errorf("Expected spy to be invoked returning 2, got %d", i)
	}
	if i := d1.call("third"); i != 3 {
		t.Errorf("Expected spy to be invoked returning 3, got %d", i)
	}

	spy := d1.Spy("call")
	spy.Expect(Exactly(3))
	assertMatch(t, spy, "all calls.*call")

	calls := spy.Matching("hello")
	calls.Expect(AtLeast(1)) //should mean doubleT gets Errorf as this expectation
	assertMatch(t, calls, `(?m)matching.*hello.*`)

	spy.Matching(func(in string) bool { return strings.HasSuffix(in, "d") }).Expect(Twice())

	spy.Slice(0, 0).Expect(Never())
	spy.Slice(5, 8).Matching(Any()).Expect(Never()) //empty set
	spy.Slice(0, 5).Expect(Exactly(3))              //even though 5 is out of range

	calls = spy.Slice(1, 2).Matching("second")
	calls.Expect(Once())
	assertMatch(t, calls, `(?s)matching.*second.*within.*\[1:2\]`)

	second := spy.Matching("second")
	spy.After(second).Expect(Once())
	calls = spy.After(second).Matching("third")
	calls.Expect(Once())
	assertMatch(t, calls, `(?s)matching.*third.*after.*matching.*second`)
	spy.After(spy.Slice(0, 0)).Expect(Exactly(3))
}

func TestTestDouble_Fake(t *testing.T) {
	d1 := newApiDouble(t)
	spy := d1.Fake("call", func(s string) int { return len(s) })
	spyEmpty := d1.Fake("empty", func() {})

	if i := d1.call("1234567"); i != 7 {
		t.Errorf("Expected fake to be invoked returning 7, got %d", i)
	}
	if i := d1.call("hello"); i != 5 {
		t.Errorf("Expected fake to be invoked returning 5, got %d", i)
	}
	d1.empty()

	spy.Expect(Twice())
	spy.Matching("hello").Expect(Once())
	spyEmpty.Expect(Once())
}

func TestTestDouble_FakeFailsFatallyForBadImplementations(t *testing.T) {
	type badInputs struct {
		name        string
		method      string
		bad         interface{}
		expectedMsg string
	}

	tests := []badInputs{
		{"NotAFunc", "call", "notAFunction", "func.*string"},
		{"NotAMethod", "nomethod", nil, "nomethod"},
		{"InvalidArgTypes", "call", func(i int) int { return 0 }, "string.*int"},
		{"TooManyArgs", "call", func(s string, i int) int { return 0 }, "expects.*1.*found.*2"},
		{"TooFewArgs", "call", func() int { return 0 }, "expects.*1.*found.*0"},
		{"InvalidReturnTypes", "call", func(i int) string { return "" }, "int.*string"},
		{"TooFewReturns", "call", func(s string) {}, `expects.*1.*found.*0`},
		{"TooManyReturns", "call", func(s string) (string, error) { return "", nil }, "expects.*1.*found.*2"},
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

			d := newApiDouble(tDouble)
			d.Fake(test.method, test.bad)
			t.Errorf("Expect unreachable")

		})
	}
}
func TestTestDouble_FakeFailsFatallyIfRegisteredAfterASpy(t *testing.T) {
	tDouble := NewTDouble(t, func(c *TestDouble) {
		//c.EnableTrace()
	})

	spy := tDouble.Fake("Fatalf", tDouble.FakeFatalf)
	defer func(spy FakeMethodCall) {
		recover()
		spy.Matching(printfMatcher(`unreachable fake`)).Expect(Once())
	}(spy)

	d := newApiDouble(tDouble)
	d.Spy("call")
	d.Fake("call", nil)
	t.Errorf("Expect unreachable")
}

func TestNewDouble_DefaultsMockNeverReturningZeroValuesForUnregisteredMethod(t *testing.T) {
	doubleT := NewTDouble(t, func(c *TestDouble) {
		//c.EnableTrace()
	})
	spy := doubleT.Spy("Errorf")
	d1 := newApiDouble(doubleT)

	if i := d1.call("unregistered"); i != 0 {
		t.Errorf("Expected 0, Got %d", i)
	}
	d1.Verify()

	spy.Matching(printfMatcher(`never`)).Expect(Once())
}

func TestTestDouble_UsesDefaultCallForUnregisteredMethod(t *testing.T) {
	d1 := newApiDouble(t, func(c *TestDouble) {
		c.SetDefaultCall(func(m Method) MethodCall {
			return m.Fake(func(s string) int { return len(s) })
		})
	})

	if i := d1.call("unregistered"); i != 12 {
		t.Errorf("Expected 12, Got %d", i)
	}

}

func TestTestDouble_UsesDefaultReturnValues(t *testing.T) {
	d1 := newApiDouble(t, func(c *TestDouble) {
		c.SetDefaultReturnValues(func(m Method) ReturnValues {
			return Values(67)
		})
	})

	if i := d1.call("unregistered"); i != 67 {
		t.Errorf("Expected 67, Got %d", i)
	}

}

func TestInvoke_TracesAllCalls(t *testing.T) {
	t.SkipNow()
}
