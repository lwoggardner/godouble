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
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

// ReturnValues implementations generate values in response to Stub, Mock or Spy method invocations
type ReturnValues interface {

	//Receive is called when a method is exercised
	//
	// non nil error response will fatally terminate the test
	Receive() ([]interface{}, error)
}

type ValidatingReturnValues interface {
	ReturnValues
	ForMethod(t T, method reflect.Method)
}

type multiValues interface {
	ReturnValues
	multiValued() bool
}

// A Timewarp can be used to simulate a sleep, eg when testing using a fake clock.
// The canonical sleeper is
//   time.After
type Timewarp func(d time.Duration) <-chan time.Time

func NewReturnsForMethod(t T, forMethod reflect.Method, values ...interface{}) (rv ReturnValues) {
	if len(values) == 1 {
		var isRv bool
		if rv, isRv = values[0].(ReturnValues); !isRv {
			rv = Values(values...)
		} //else rv is now cast to ReturnValues
	} else {
		rv = Values(values...)
	}
	if validatingRV, hasForMethod := rv.(ValidatingReturnValues); hasForMethod {
		validatingRV.ForMethod(t, forMethod)
	}
	return
}

type reflectZeroReturnValues []reflect.Type

func (zv reflectZeroReturnValues) Receive() ([]interface{}, error) {
	if len(zv) == 0 {
		return nil, nil
	}
	results := make([]interface{}, len(zv))
	for i := 0; i < len(zv); i++ {
		results[i] = reflect.Zero(zv[i]).Interface()
	}
	return results, nil
}

func (zv reflectZeroReturnValues) ForMethod(_ T, _ reflect.Method) {
	//we validated on the way in
}

// ZeroValues repeatedly returns the zeroed values for the given methodType
func ZeroValues(methodType reflect.Type) ReturnValues {
	if methodType.NumOut() == 0 {
		return reflectZeroReturnValues(nil)
	}
	results := make([]reflect.Type, methodType.NumOut())
	for i := 0; i < methodType.NumOut(); i++ {
		results[i] = methodType.Out(i)
	}
	return reflectZeroReturnValues(results)
}

type fixedReturnValues []interface{}

func (v fixedReturnValues) Receive() ([]interface{}, error) {
	return v, nil
}
func (v fixedReturnValues) ForMethod(t T, m reflect.Method) {
	AssertMethodReturnValues(t, m, v)
}

// Values stores a fixed set of values returned for every invocation
func Values(values ...interface{}) ReturnValues {
	return fixedReturnValues(values)
}

// ReturnChannel provides channel semantics for returning values from stub calls
type ReturnChannel interface {

	//Send a list of return values
	Send(...interface{})

	//Close the channel, subsequent invocations that need values will cause the test to fail fatally
	Close()

	//Set a timeout. If the timeout expires before a Value is available on the channel
	//  ( via Send() ) the test will fail fatally.
	SetTimeout(timeout time.Duration, sleeper ...Timewarp)

	ReturnValues
}

// NewReturnChannel generates return values for successive calls to a stub.
// It will return errors if the channel is closed
//
// Use the optional bufferSize parameter with a non-zero Value to create a buffered channel.
//
// Use SetTimeout() to override the default timeout of 200 ms.
func NewReturnChannel(bufferSize ...int) ReturnChannel {
	var channel chan []interface{}

	bufSize := 0
	for _, size := range bufferSize {
		bufSize += size
	}
	channel = make(chan []interface{}, bufSize)
	return &returnChannel{
		values:  channel,
		timeout: 200 * time.Millisecond,
		sleeper: time.After,
	}

}

type returnChannel struct {
	t       T
	method  reflect.Method
	values  chan []interface{}
	timeout time.Duration
	sleeper Timewarp
}

func (rc *returnChannel) ForMethod(t T, method reflect.Method) {
	rc.t = t
	rc.method = method
}

func (rc *returnChannel) multiValued() bool { return true }

func (rc *returnChannel) Receive() (returns []interface{}, err error) {
	select {
	case generatedReturns, ok := <-rc.values:
		if ok {
			returns = generatedReturns
		} else {
			err = errors.New("requested values from closed return channel")
		}
	case _ = <-rc.sleeper(rc.timeout):
		err = errors.New("timed out waiting for return channel to provide values")
	}

	return
}

func (rc *returnChannel) Send(returnValues ...interface{}) {
	if rc.t != nil {
		AssertMethodReturnValues(rc.t, rc.method, returnValues)
	}
	rc.values <- returnValues
}

func (rc *returnChannel) Close() {
	close(rc.values)
}

//Max time to wait for a Value from the channel before failing the test
func (rc *returnChannel) SetTimeout(timeout time.Duration, sleeper ...Timewarp) {
	if len(sleeper) > 0 {
		rc.sleeper = sleeper[0]
	}
	rc.timeout = timeout
}

type delayedReturnValues struct {
	ReturnValues
	delayer func() time.Duration
	sleeper Timewarp
}

func newDelayedReturnValues(rv ReturnValues, f func() time.Duration, sleeper ...Timewarp) ReturnValues {
	sf := time.After
	if len(sleeper) > 0 {
		sf = sleeper[0]
	}
	return &delayedReturnValues{ReturnValues: rv, delayer: f, sleeper: sf}
}

func (d *delayedReturnValues) Receive() ([]interface{}, error) {
	//Simulate IO delay / long poll etc
	<-d.sleeper(d.delayer())
	return d.ReturnValues.Receive()
}

func (d delayedReturnValues) ForMethod(t T, method reflect.Method) {
	if rvForMethod, hasForMethod := d.ReturnValues.(ValidatingReturnValues); hasForMethod {
		rvForMethod.ForMethod(t, method)
	}
}

// Delayed wraps the ReturnValues rv with a fixed delay of 'by' duration
//
// Useful to simulate an asynchronous IO request, allowing other goroutines to run
// while waiting for the response.
//
// An optional sleeper function, defaulting to time.Sleep, can be provided. eg for use with fake clock
func Delayed(rv ReturnValues, by time.Duration, sleep ...Timewarp) ReturnValues {
	return newDelayedReturnValues(rv, func() time.Duration { return by }, sleep...)
}

// RandDelayed wraps the ReturnValues rv with a delay of up to 'max' duration
func RandDelayed(rv ReturnValues, max time.Duration, sleep ...Timewarp) ReturnValues {
	return newDelayedReturnValues(rv, func() time.Duration { return time.Duration(rand.Int63n(int64(max))) }, sleep...)
}

type sequentialReturnValues struct {
	values []ReturnValues
	rvChan <-chan []interface{}
	once   *sync.Once
}

func (s *sequentialReturnValues) Receive() (returns []interface{}, err error) {
	s.once.Do(s.run)
	if generatedReturns, ok := <-s.rvChan; ok {
		returns = generatedReturns
	} else {
		err = errors.New("no available values")
	}
	return
}

func (s *sequentialReturnValues) multiValued() bool { return true }

//Sequence returns values from each of 'values' until there are no further values available
func Sequence(values ...ReturnValues) ReturnValues {
	return &sequentialReturnValues{values: values, once: &sync.Once{}}
}

func (s *sequentialReturnValues) ForMethod(t T, m reflect.Method) {
	for _, rv := range s.values {
		if validatingRV, isValidating := rv.(ValidatingReturnValues); isValidating {
			validatingRV.ForMethod(t, m)
		}
	}
}

func (s *sequentialReturnValues) run() {
	rvChan := make(chan []interface{})
	s.rvChan = rvChan
	go func(s *sequentialReturnValues) {
		for _, rv := range s.values {
			if mv, isMultiValue := rv.(multiValues); isMultiValue && mv.multiValued() {
				for {
					if result, err := mv.Receive(); err != nil {
						break
					} else {
						rvChan <- result
					}
				}
			} else if result, err := rv.Receive(); err == nil {
				rvChan <- result
			}
		}
		close(rvChan)

	}(s)
}
