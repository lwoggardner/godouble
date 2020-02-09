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

//AssertMethodReturnValues fatally fails test t unless returnValues are compatible with method
func AssertMethodReturnValues(t T, method reflect.Method, returnValues []interface{}) {
	t.Helper()
	returnTypes := make([]reflect.Type, len(returnValues))
	for i, v := range returnValues {
		returnTypes[i] = reflect.TypeOf(v)
	}
	AssertMethodReturnTypes(t, method, returnTypes, method.Type, " ")
}

//AssertMethodOutputs Fatally fails test t unless funcType's return types are compatible with m's return types
func AssertMethodOutputs(t T, m reflect.Method, funcType reflect.Type) {
	t.Helper()
	if funcType.Kind() != reflect.Func {
		t.Fatalf("expected func, got %v", funcType)
	}

	if m.Type.NumOut() != funcType.NumOut() {
		t.Fatalf("%v for %v expects to have %d return values, found %d", funcType, m.Type, m.Type.NumOut(), funcType.NumOut())
	}

	returnTypes := make([]reflect.Type, funcType.NumOut())
	for i := 0; i < funcType.NumOut(); i++ {
		returnTypes[i] = funcType.Out(i)
	}
	AssertMethodReturnTypes(t, m, returnTypes)
}

//AssertMethodReturnTypes fatally fails test t unless returnTypes are compatible with method m's return types
func AssertMethodReturnTypes(t T, m reflect.Method, returnTypes []reflect.Type, prefixes ...interface{}) {
	t.Helper()
	if m.Type.NumOut() != len(returnTypes) {
		t.Fatalf("%v for %sexpects to have %d return values, found %d", m.Type, fmt.Sprint(prefixes...), m.Type.NumOut(), len(returnTypes))
	}

	for i, out := range returnTypes {
		if mType := m.Type.Out(i); out != nil && !out.AssignableTo(mType) {
			t.Fatalf("%v for %sexpects to have return Value %d to be assignable to %v, got %v", m.Type, fmt.Sprint(prefixes...), i, mType, out)
		}
	}
}

//AssertMethodInputs fatally fails test t unless funcType has compatible input methods with method m
func AssertMethodInputs(t T, m reflect.Method, funcType reflect.Type) {
	t.Helper()
	if funcType.Kind() != reflect.Func {
		t.Fatalf("expected func, got %v", funcType)
	}

	if funcType.IsVariadic() != m.Type.IsVariadic() {
		t.Fatalf("%v expects %v to have variadic=%v, found %v", m.Type, funcType, m.Type.IsVariadic(), funcType.IsVariadic())
	}

	if funcType.NumIn() != m.Type.NumIn() {
		t.Fatalf("%v expects %v to have %d arguments, found %d", m.Type, funcType, m.Type.NumIn(), funcType.NumIn())
	}

	for i := 0; i < funcType.NumIn(); i++ {
		if !m.Type.In(i).AssignableTo(funcType.In(i)) {
			t.Fatalf("%v requires %v arg %d to be assignable from %v", m.Type, funcType, i, m.Type.In(i))
		}
	}
}
