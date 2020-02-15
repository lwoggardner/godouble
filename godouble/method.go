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
	"sync"
)

// Method is used to configure the default Double type for a given interface method.
//
// A method's signature is available via Reflect()
// Stub(), Mock(), Spy(), Fake() are used to return a specific MethodCall implementation to use
// See TestDoubleConfigurator
type Method interface {
	Stub() StubbedMethodCall
	Mock() MockedMethodCall
	Spy() SpyMethodCall
	Fake(impl interface{}) FakeMethodCall
	Reflect() reflect.Method
}

type method struct {
	receiver *TestDouble
	mutex    *sync.Mutex
	calls    []MethodCall
	m        reflect.Method
}

func newMethod(d *TestDouble, m reflect.Method) *method {
	return &method{d, &sync.Mutex{}, []MethodCall{}, m}
}

func (m *method) trace() bool {
	return m.receiver.trace
}

func (m *method) t() T {
	return m.receiver.t
}
func (m *method) Stub() StubbedMethodCall {
	return newStubbedMethodCall(m)
}

func (m *method) Mock() MockedMethodCall {
	return newMockedMethodCall(m)
}

func (m *method) Spy() SpyMethodCall {
	return newSpyMethodCall(m)
}

func (m *method) Fake(impl interface{}) FakeMethodCall {
	return newFakeMethodCall(m, impl)
}

func (m *method) Reflect() reflect.Method {
	return m.m
}

func (m *method) String() string {
	return fmt.Sprintf("%v.%s", m.receiver, m.m.Name)
}

func (m *method) addMethodCall(call MethodCall) {
	m.calls = append(m.calls, call)
}

func (m *method) match(args []interface{}) (matched MethodCall) {
	for _, possible := range m.calls {
		if possible.matches(args) {
			return possible
		}
	}
	defaultMatcher := m.receiver.defaultCall(m)
	if defaultMatcher == nil {
		m.t().Fatalf("Nil DefaultMethodCall returned for %v", m)
	} else if !defaultMatcher.matches(args) {
		m.t().Fatalf("Method %v expects default matcher %v to match %v", m, matched, args)
	}
	m.addMethodCall(defaultMatcher)

	return defaultMatcher
}
func (m *method) invoke(args []interface{}) []interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	matched := m.match(args)

	if m.trace() {
		m.t().Helper()
		//A fake method can panic but we still want to trace it
		defer func(matched MethodCall, args []interface{}) {
			if e := recover(); e != nil {
				m.t().Logf("Called %s(%v) => panic! %v", matched, args, e)
				panic(e)
			}
		}(matched, args)
	}

	returns, err := matched.spy(args)
	if err != nil {
		m.t().Fatalf("No return values available for method %v(%v) %s", matched, args, err.Error())
	} else {
		if m.trace() {
			m.t().Logf("Called %s(%v) => %v", matched, args, returns)
		}
		AssertMethodReturnValues(m.t(), m.m, returns) //Safe but slow?
	}
	return returns
}

func (m *method) defaultReturnValues() ReturnValues {
	return m.receiver.defaultReturnValues(m)
}
