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

/*
Package godouble is a TestDouble framework for Go.

This framework creates a TestDouble implementation of an interface which can be substituted for the real thing during
tests. Interface methods can then individually be Stubbed, Mocked, Spied upon or Faked as required.

Stubs, Mocks, Spies, Fakes

See the canonical sources...

* http://xunitpatterns.com/Test%20Double.html

* https://martinfowler.com/articles/mocksArentStubs.html


A Stub provides specific return values for a matching call to the method.  Most useful where the return values
are the primary means by which correct operation of the system under test can be verified.

 package examples

 import (
	. "github.com/lwoggardner/godouble" //Note the dot import which assists with readability
	"testing"
 )

 func Test_Stub(t *testing.T) {
	d := NewAPIDouble(t) // A specific implementation of a TestDouble

	//Stub a method that receives specific arguments, to return specific values
	d.Stub("SomeQuery").Matching(Args(Eql("test"))).Returning(Values(Results{"result"}, nil))

	// Exercise the system under test substituting d for the real API client
	// ...

	// Verify assertions to confirm the system under test behaves as expected with the given return values
	// ...
 }


A Mock is a Stub with an up-front expectation for how many times it will be called.
Most useful when the return values of the method do not completely ensure correct functioning of the system under test,


 func Test_Mock(t *testing.T) {
	d := NewAPIDouble(t)
    // Verify the mock expectations are met at completion
	defer d.Verify()

	//Stub a method that receives specific arguments, returns specific values and explicitly expects to be called once
	d.Mock("SomeQuery").Matching(Args(Eql("test"))).Returning(Values(Results{"result"}, nil)).Expect(Exactly(3))
	d.Mock("OtherMethod").Expect(Never())

    //Exercise...
 }


A Spy is a record of all calls made to a method which can be verified after exercising the system under test.
Used similarly to Mock, but where you prefer to explicitly assert received arguments and call counts in the Verify phase
of the test.


 func Test_Spy(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)

	spy := d.Spy("SomeQuery").Returning(Values(Results{"nothing"}, nil))

	//Exercise...

	//Verify
    spy.Expect(Twice()) //All calls
	spy.Matching(Args(Eql("test"))).Expect(Once()) //The subset of calls with matching args

 }


A Fake is a Spy that provides an actual implementation of the method instead of return values. Use with caution.

 func Test_Fake(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)
	impl := func( i int, options...string) *Results {
		return &Results{Output: fmt.Sprintf("%s %d",strings.Join(options," "),i)}
	}

	spy := d.Fake("QueryWithOptions",impl)

	//Exercise...

	//Verify
	spy.Expect(Twice())
	spy.Matching(Args(Eql(10))).Expect(Once())

 }

*/
package godouble
