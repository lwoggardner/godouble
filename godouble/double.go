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
)

//T is compatible with builtin testing.T
type T interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
	Helper()
}

//MatcherForMethod can be used to integrate a different matching framework
type MatcherForMethod func(t T, m reflect.Method, chained MethodArgsMatcher, matchers ...interface{}) MethodArgsMatcher

//ReturnsForMethod can be used to integrate a different return values framework
type ReturnsForMethod func(t T, m reflect.Method, chained ReturnValues, returnValues ...interface{}) ReturnValues

/*  We could use this to determine what phase we are in as some methods are only valid in Setup Phase
type TestPhase int
const (
	SetupPhase TestPhase = iota
	ExercisePhase
	VerifyPhase
	TeardownPhase
)
*/

/*
A TestDouble is an object that can substitute for a concrete implementation of an interface
in a 4 phase testing framework (Setup, Exercise, Verify, Teardown).

Setup phase

Expected method calls to the double can be configured as one of the following types.

1) Stub - Returns known values in response to calls against matching input arguments

2) Mock - A stub with pre-built expectations about the number and order of method invocations on matching calls

3) Spy  - A stub that records calls as they execute

4) Fake - A substitute implementation for the method

Exercise phase

Any methods invoked on the double are sent to the first matching call that has been configured. If no matching call
is available, the DefaultMethodCallType for this double is generated.

Verify phase

The Verify() method is used to confirm expectations on Mock methods have been met.

Spies (and Fakes) have explicit methods to assert the number and order of method invocations on subsets of calls.
*/
type TestDouble struct {
	t                   T
	methods             map[string]*method
	defaultCall         func(Method) MethodCall
	defaultReturnValues func(Method) ReturnValues
	forInterface        reflect.Type
	trace               bool
	matcher             MatcherForMethod
	returns             ReturnsForMethod
}

// Enable tracing of all received method calls (via T.Logf)
func (d *TestDouble) EnableTrace() {
	d.trace = true
}

/*
SetDefaultCall allows caller to provide a function to decide whether to Stub, Mock, Spy or Fake
a call that was not explicitly registered in Setup phase.

the default function is a mock that never expects to be called.
*/
func (d *TestDouble) SetDefaultCall(defaultCall func(Method) MethodCall) {
	d.defaultCall = defaultCall
}

/*
	SetDefaultReturnValues allows a caller to provide a function to generate default return values
	for a Stub, Mock, or Spy that was not explicitly registered with ReturnValues during Setup.
	The default is to used zeroed values via reflection.
*/
func (d *TestDouble) SetDefaultReturnValues(defaultReturns func(Method) ReturnValues) {
	d.defaultReturnValues = defaultReturns
}

func (d *TestDouble) SetMatcherIntegration(forMethod MatcherForMethod) {
	d.matcher = forMethod
}

func (d *TestDouble) SetReturnValuesIntegration(forMethod ReturnsForMethod) {
	d.returns = forMethod
}

func (d *TestDouble) String() string {
	return fmt.Sprintf("DoubleFor(%v)", d.forInterface)
}

func (d *TestDouble) T() T {
	return d.t
}

//MethodCall is an abstract interface of specific call types, Stub, Mock, Spy and Fake
type MethodCall interface {
	matches(args []interface{}) bool
	spy(args []interface{}) ([]interface{}, error)
	verify(T)
}

/*
NewDouble Constructor for TestDouble called by specific implementation of test doubles.

forInterface is expected to be the nil implementation of an interface - (*Iface)(nil)

configurators are used to configure tracing and default behaviour for unregistered method calls and return values
*/
func NewDouble(t T, forInterface interface{}, configurators ...func(*TestDouble)) *TestDouble {
	doubleFor := reflect.TypeOf(forInterface)

	if doubleFor.Kind() != reflect.Ptr || doubleFor.Elem().Kind() != reflect.Interface {
		t.Fatalf("Expecting '%v' to be a pointer to nil interface", forInterface)
	}
	doubleFor = doubleFor.Elem()

	double := &TestDouble{
		t:            t,
		forInterface: doubleFor,
		methods:      make(map[string]*method, doubleFor.NumMethod()),
	}

	for i := 0; i < doubleFor.NumMethod(); i++ {
		m := doubleFor.Method(i)
		double.methods[m.Name] = newMethod(double, m)
	}

	defaults(double)
	for _, c := range configurators {
		c(double)
	}

	if double.matcher == nil {
		t.Fatalf("%v need SetMatcherIntegration() configured", doubleFor)
	}

	if double.returns == nil || double.defaultReturnValues == nil {
		t.Fatalf("%v needs both SetReturnValuesIntegration and SetDefaultReturnValues configured", doubleFor)
	}

	if double.defaultCall == nil {
		t.Fatalf("%v needs SetDefaultCall configured", doubleFor)
	}

	return double
}

/*
Stub adds and returns a StubbedMethodCall for methodName on TestDouble d

Setup phase

Configure Matcher and ReturnValues.

By default a StubbedMethodCall matches any arguments and returns zero values for all outputs.

Exercise Phase

The first stub matching the invocation arguments will provide the output values.

Verify Phase

Nothing to verify
*/
func (d *TestDouble) Stub(methodName string) (stub StubbedMethodCall) {

	if m, found := d.methods[methodName]; found {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		stub = m.Stub()
		m.addMethodCall(stub)
	} else {
		d.t.Fatalf("Cannot Stub non existent method %s for %v", methodName, d)
	}
	return
}

/*
Mock adds and returns a MockedMethodCall for methodName on TestDouble d

Setup Phase

Configure Matcher, sequencing (After), and Return Values.

Set Expectation on number of matching invocations.

By default a MockedMethodCall matches any arguments, returns zero values for all outputs and
expects exactly one invocation.

Exercise Phase

The first mock matching the invocation arguments and not yet Complete in terms of Expectation will
provide the output values.

Verify Phase

(via call to a TestDouble.Verify() usually deferred immediately after the double is created)

Will assert the Expectation is met.

*/
func (d *TestDouble) Mock(methodName string) (mock MockedMethodCall) {
	if m, found := d.methods[methodName]; found {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		mock = m.Mock()
		m.addMethodCall(mock)
	} else {
		d.t.Fatalf("Cannot Mock non existent method %s for %v", methodName, d)
	}
	return
}

/*
Spy records all calls to methodName.

Setup Phase

Configure ReturnValues.

Calling Spy twice for the same method will return the same Value (ie there is only every one spy,
and it will record methods that do not match any preceding Stub or Mock calls)

Exercise Phase

Matches and records all invocations.

Verify Phase

Can be called again to retrieve the spy for the method (eg to get a dynamically created default Spy).

Extract subsets of RecordedCalls and then verify an Expectations on the number of calls in the subset.

*/
func (d *TestDouble) Spy(methodName string) (spy SpyMethodCall) {
	if m, found := d.methods[methodName]; found {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		for _, methodCall := range m.calls {
			if call, isa := methodCall.(SpyMethodCall); isa {
				return call
			}
		}
		spy = m.Spy()
		m.addMethodCall(spy)
	} else {
		d.t.Fatalf("Cannot Spy on non existent method %s for %v", methodName, d)
	}
	return
}

/*
Fake installs a user implementation for the method.

Setup Phase

Install the Fake implementation, which must match the signature of the method.

Only one fake is installed for a method, and clobbers any other configured calls.

Exercise Phase

Invokes the fake function via reflection, and records the call as per Spy.

Verify Phase

Explicitly verify RecordedCalls as per Spy.
*/
func (d *TestDouble) Fake(methodName string, impl interface{}) (fake FakeMethodCall) {

	if m, found := d.methods[methodName]; found {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		for _, methodCall := range m.calls {
			if call, isa := methodCall.(SpyMethodCall); isa {
				d.t.Fatalf("unreachable fake for %s.%s which has previously registered a spy (%v)", d, methodName, call)
			}
		}
		fake = m.Fake(impl)
		m.addMethodCall(fake)
	} else {
		d.t.Fatalf("Cannot Fake non existent method %v.%s", d, methodName)
	}
	return
}

func (d *TestDouble) Verify() {
	for _, method := range d.methods {
		for _, methodCall := range method.calls {
			methodCall.verify(d.t)
		}
	}
}

//Invoke is called by specialised mock implementations, and sometimes by Fake implementations
//to record the invocation of a method.
func (d *TestDouble) Invoke(methodName string, args ...interface{}) []interface{} {
	d.t.Helper()

	method, found := d.methods[methodName]
	if !found {
		d.t.Fatalf("Unexpected call to unknown methodName %T.%s", d, methodName)
	}
	return method.invoke(args)
}

type Verifiable interface {
	Verify()
}

//Verify is shorthand to Verify a set of TestDoubles
func Verify(testDoubles ...Verifiable) {
	for _, td := range testDoubles {
		td.Verify()
	}
}
