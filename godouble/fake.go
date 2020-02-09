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
	"reflect"
)

// FakeMethodCall is a SpyMethodCall with a Fake implementation.
type FakeMethodCall interface {
	// The full set of all recorded calls to this method available to be verified
	RecordedCalls
	MethodCall
}

type fakeMethodCall struct {
	*spyMethodCall
	impl reflect.Value
}

func newFakeMethodCall(m *method, impl interface{}) *fakeMethodCall {

	implF := reflect.ValueOf(impl)
	implT := implF.Type()
	AssertMethodInputs(m.t(), m.Reflect(), implT)
	AssertMethodOutputs(m.t(), m.Reflect(), implT)

	return &fakeMethodCall{spyMethodCall: newSpyMethodCall(m), impl: implF}
}

func (c *fakeMethodCall) spy(args []interface{}) ([]interface{}, error) {
	//Record the call first, in case the actual call panics.
	c.recorded = append(c.recorded, newRecordedCall(args))

	inArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		inArgs[i] = reflect.ValueOf(arg)
	}
	var returnVals []reflect.Value
	if c.impl.Type().IsVariadic() {
		returnVals = c.impl.CallSlice(inArgs)
	} else {
		returnVals = c.impl.Call(inArgs)
	}

	if len(returnVals) == 0 {
		return nil, nil
	}
	returns := make([]interface{}, len(returnVals))
	for j, v := range returnVals {
		returns[j] = v.Interface()
	}
	return returns, nil
}
