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
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestReturnValues(t *testing.T) {

	apiCallMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("call")
	emptyMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("empty")
	multiMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("test")
	type test struct {
		name string
		ReturnValues
		method   reflect.Method
		expected []interface{}
	}

	tests := []test{
		{"WithSingleValue", Values(10), apiCallMethod, []interface{}{10}},
		{"WithMultipleValues", Values(10, errors.New("xxx")), multiMethod, []interface{}{10, errors.New("xxx")}},
		{"WithNoValues", Values(), emptyMethod, nil},
		{"WithZeroValues", ZeroValues(apiCallMethod.Type), apiCallMethod, []interface{}{0}},
		{"WithNoZeroValues", ZeroValues(emptyMethod.Type), emptyMethod, nil},
	}

	for _, test := range tests {
		values := test
		t.Run(values.name, func(t *testing.T) {
			NewReturnsForMethod(t, values.method, values.ReturnValues)
			returns, err := values.Receive()
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
			if !reflect.DeepEqual(returns, values.expected) {
				t.Errorf("Expected %v returns, got %v", values.expected, returns)
			}
		})
	}
}

func TestReturnValues_FatallyFailsTheTest(t *testing.T) {
	apiCallMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("call")
	type test struct {
		name string
		ReturnValues
		method      reflect.Method
		expectedMsg string
	}

	testTable := []test{
		{"WithIncorrectTypes", Values("astring"), apiCallMethod, "int.*string"},
		{"WithTooFewValues", Values(), apiCallMethod, "expects.* 1.*found 0"},
		{"WithTooManyValues", Values(10, "extra"), apiCallMethod, "expects.* 1.*found 2"},
	}

	for _, test := range testTable {
		values := test
		t.Run(values.name, func(t *testing.T) {
			tDouble := NewTDouble(t, func(c *TestDouble) {
				//c.EnableTrace()
			})
			spy := tDouble.Fake("Fatalf", tDouble.FakeFatalf)
			defer func(spy FakeMethodCall) {
				recover()
				spy.Matching(printfMatcher(values.expectedMsg)).Expect(Once())
			}(spy)

			NewReturnsForMethod(tDouble, values.method, values.ReturnValues)
		})
	}
}

func TestDelayed(t *testing.T) {
	apiCallMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("call")

	delay := time.Duration(60) * time.Millisecond
	delayed := NewReturnsForMethod(t, apiCallMethod, Delayed(Values(55), delay))
	before := time.Now()
	returns, err := delayed.Receive()
	if len(returns) != 1 || err != nil || returns[0].(int) != 55 {
		t.Errorf("Expected received values [55], got %v", returns)
	}
	after := time.Now()
	actualDelay := after.Sub(before)
	maxExpectedDelay := delay + (time.Duration(10) * time.Millisecond)
	if actualDelay < delay || actualDelay > maxExpectedDelay {
		t.Errorf("Expected delay to be within 20ms of %v, actual delay %v", delay, actualDelay)
	}
}

func TestDelayedWithSleepFunc(t *testing.T) {
	rv := Values(99)

	delay := time.Duration(60) * time.Millisecond
	var received time.Duration
	delayed := Delayed(rv, delay, func(d time.Duration) <-chan time.Time { received = d; return time.After(0) })
	_, _ = delayed.Receive()
	if received != delay {
		t.Errorf("Expected sleep function to receive %v, got %v", delay, received)
	}
}

func TestRandDelayed(t *testing.T) {
	rv := Values(33)

	delay := time.Duration(600) * time.Millisecond
	var received time.Duration
	delayed := RandDelayed(rv, delay, func(d time.Duration) <-chan time.Time { received = d; return time.After(0) })
	for i := 0; i < 100; i++ {
		_, _ = delayed.Receive()
		if received >= delay {
			t.Errorf("Expected iteration %d, sleep function to receive a random Value less than %v, got %v", i, delay, received)
		}
	}
}

func TestReturnChannel(t *testing.T) {
	type returnChannelTest struct {
		name     string
		toSend   []interface{}
		method   reflect.Method
		sleeper  Timewarp
		buffered bool
	}

	apiCallMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("call")

	sender := func(rc ReturnChannel, values []interface{}) {
		for _, v := range values {
			rc.Send(v)
		}
	}

	fakeTimeout := func(d time.Duration) <-chan time.Time {
		if d != time.Duration(20)*time.Millisecond {
			t.Errorf("Expected duration 20ms, got %v", d)
		}
		return time.After(0)
	}

	tests := []returnChannelTest{
		{"SendBuffered", []interface{}{10, 15, -1}, apiCallMethod, nil, true},
		{"SendUnbuffered", []interface{}{3, 2}, apiCallMethod, nil, false},
		{"CloseWithoutSend", []interface{}{}, apiCallMethod, fakeTimeout, true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var rc ReturnChannel
			if test.buffered {
				rc = NewReturnChannel(len(test.toSend))
			} else {
				rc = NewReturnChannel()
			}
			NewReturnsForMethod(t, test.method, rc)

			if test.buffered {
				sender(rc, test.toSend)
			} else {
				go sender(rc, test.toSend)
			}

			for i, v := range test.toSend {
				values, err := rc.Receive()
				if err != nil {
					t.Errorf("Expecting non nil error from Receive, got %v", err)
				} else {
					received, _ := values[0].(int)
					if received != v {
						t.Errorf("Expected received[%d] Value %d, got %d", i, v, received)
					}
				}
			}

			//Expect timeout on next receive
			if test.sleeper != nil {
				rc.SetTimeout(time.Duration(20)*time.Millisecond, test.sleeper)
			}
			_, err := rc.Receive()
			if err == nil {
				t.Errorf("Expected error on receive from channel with nothing sending")
			} else if matched, _ := regexp.MatchString("timed out", err.Error()); !matched {
				t.Errorf("Expected %s to match `timed out`", err.Error())
			}

			rc.Close()
			_, err = rc.Receive()
			if err == nil {
				t.Errorf("Expected error on receive from closed channel")
			} else if matched, _ := regexp.MatchString("closed.*channel", err.Error()); !matched {
				t.Errorf("Expected %s to match `closed.*channel`", err.Error())
			}
		})
	}

}

func TestSequence(t *testing.T) {
	type test struct {
		name     string
		values   []ReturnValues
		expected []int
	}
	apiCallMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("call")

	rc := NewReturnChannel(2)
	rc.Send(11)
	rc.Send(12)
	rc.Close()
	tests := []test{
		{"NoValues", []ReturnValues{Values(33), Values(44)}, []int{33, 44}},
		{"SeqOfSeq", []ReturnValues{Sequence(Values(33), Values(55)), Values(44)}, []int{33, 55, 44}},
		{"MultiSeq", []ReturnValues{Sequence(Values(33), Values(55)), Sequence(Values(44), Values(66))}, []int{33, 55, 44, 66}},
		{"WithChannel", []ReturnValues{rc, Values(44)}, []int{11, 12, 44}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			seq := Sequence(test.values...).(*sequentialReturnValues)
			seq.ForMethod(t, apiCallMethod)
			for _, ex := range test.expected {
				rcv, err := seq.Receive()
				if err != nil || len(rcv) != 1 {
					t.Errorf("Expected [1]int, nil got %v,%v", rcv, err)
				} else if actual, ok := rcv[0].(int); !ok {
					t.Errorf("Got expected int, got %v", rcv[0])
				} else if actual != ex {
					t.Errorf("expected %d, got %d", ex, actual)
				}
			}
		})
	}
}

func TestReturnChannel_SendFatallyFailsTheTest(t *testing.T) {
	apiCallMethod, _ := reflect.TypeOf((*api)(nil)).Elem().MethodByName("call")
	type test struct {
		name        string
		values      []interface{}
		method      reflect.Method
		expectedMsg string
	}

	tests := []test{
		{"WithIncorrectTypes", []interface{}{"astring"}, apiCallMethod, "int.*string"},
		{"WithTooFewValues", nil, apiCallMethod, "expects.* 1.*found 0"},
		{"WithTooManyValues", []interface{}{10, "extra"}, apiCallMethod, "expects.* 1.*found 2"},
	}

	for _, test := range tests {
		values := test
		t.Run(values.name, func(t *testing.T) {
			tDouble := NewTDouble(t, func(c *TestDouble) {
				//c.EnableTrace()
			})
			spy := tDouble.Fake("Fatalf", tDouble.FakeFatalf)
			defer func(spy FakeMethodCall) {
				recover()
				spy.Matching(printfMatcher(values.expectedMsg)).Expect(Once())
			}(spy)

			rc := NewReturnChannel()
			defer rc.Close()
			newRC := NewReturnsForMethod(tDouble, values.method, rc)
			if newRC != rc {
				t.Errorf("Expected %v=%v", newRC, rc)
			}
			rc.Send(values.values...)
		})
	}
}
