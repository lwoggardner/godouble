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

package examples

//go:generate go run -tags doublegen doublegen/example_gen.go

import (
	"fmt"
	. "github.com/lwoggardner/godouble/godouble" //Note the dot import which assists with readability
	"strings"
	"testing"
)

func Test_Mock(t *testing.T) {
	d := NewAPIDouble(t)
	defer d.Verify()

	//Setup
	d.Mock("SomeCommand").Expect(Exactly(3))

	//Exercise
	d.SomeCommand()
	d.SomeCommand()
	d.SomeCommand()

	//Verify (deferred)
}

func Test_Stub(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)

	d.Stub("SomeQuery").Matching(Args(Eql("other"))).Returning(Values(Results{"nothing"}, nil))

	d.Stub("SomeQuery").Matching(Args(Eql("test"))).Returning(Values(Results{"result"}, nil))

	//Exercise
	r, e := d.SomeQuery("test")

	//Verify
	if e != nil {
		t.Errorf("Expecting nil error, got %v", e)
	}
	if r.Output != "result" {
		t.Errorf("Expecting 'result', Got '%s'", r.Output)
	}

}

func Test_Spy(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)

	spy := d.Spy("SomeQuery").Returning(Values(Results{"nothing"}, nil))

	//Exercise
	r, e := d.SomeQuery("test")

	//Verify
	spy.Expect(Once())

	if e != nil {
		t.Errorf("Expecting nil error, got %v", e)
	}
	if r.Output != "nothing" {
		t.Errorf("Expecting 'nothing', Got '%s'", r.Output)
	}

}

func Test_Fake(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)
	fake := func(i int, options ...string) *Results {
		return &Results{Output: fmt.Sprintf("%s %d", strings.Join(options, " "), i)}
	}

	spy := d.Fake("QueryWithOptions", fake)

	//Exercise
	d.QueryWithOptions(5)
	r := d.QueryWithOptions(10, "hello", "fake")

	//Verify
	//spy.Matching returns a subset of calls to the spy (in this case a fake)
	//which can then be validated in terms of the number.
	spy.Expect(Twice())
	spy.Matching(Args(Eql(10))).Expect(Once())
	if r.Output != "hello fake 10" {
		t.Errorf("Expected 'hello fake 10', Got %s", r.Output)
	}
}

func Test_ReturnChannel(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)
	returns := NewReturnChannel()
	d.Stub("SomeQuery").Returning(returns)
	go func() {
		returns.Send(Results{Output: "One"}, nil)
		returns.Send(Results{Output: "Two"}, nil)
		returns.Close()
	}()

	//Exercise
	r1, _ := d.SomeQuery("1")
	r2, _ := d.SomeQuery("2")

	//Verify
	if r1.Output != "One" {
		t.Errorf("Expected 'One', Got %s", r1.Output)
	}
	if r2.Output != "Two" {
		t.Errorf("Expected 'Two', Got %s", r2.Output)
	}
}

func Test_Unexported(t *testing.T) {
	d := NewAPIDouble(t)
	d.Stub("local").Returning(Values(exampleint(1)))
	r := d.local(99)
	if r != exampleint(1) {
		t.Errorf("Expected '1', Got %d", int(r))
	}
}
