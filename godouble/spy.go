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
	"sort"
	"strings"
	"sync/atomic"
)

var tick uint64 //global atomic counter to assist with verifying order of execution

// SpyMethodCall is a MethodCall that records method invocations for later verification
type SpyMethodCall interface {
	/*
		Returning is used to setup return values for this call

		The returnValues are converted to a ReturnValues via Values()
	*/
	Returning(values ...interface{}) SpyMethodCall

	// The full set of all recorded calls to this method available to be verified
	RecordedCalls

	MethodCall
}

// RecordedCalls represents a set of recorded call invocations to be verified
type RecordedCalls interface {
	/*
		Matching returns the subset of calls that match

		Empty matcherList list will fatally fail the test

		If the first matcher is a Matcher then it is used (test will fatally fail is more matcherList are sent)
		If the first matcher is a func then is equivalent to Matching(Matcher(matcherList[0],matcherList[1:))
		Otherwise each matcher is converted to a Matcher via either Func() or Eql()
		and this list is sent to Args()
	*/
	Matching(matchers ...interface{}) RecordedCalls

	/*
		Slice returns a subset of these calls, including call at index from, excluding call at index to (like go slice))

		If necessary use NumCalls() to reference calls from the end of the slice.
		eg to get the last 3 calls - r.Slice(r.NumCalls() -3, r.NumCalls())
	*/
	Slice(from int, to int) RecordedCalls

	// After returns the subset of these calls that were invoked after all of otherCalls
	After(otherCalls RecordedCalls) RecordedCalls

	// Expect asserts the number of calls in this set
	Expect(expect Expectation)

	// NumCalls returns the number of calls in this set.
	// Prefer to use Expect() rather than asserting the result of NumCalls()
	NumCalls() int

	calls() []*recordedCall
	nested() []string
}

type recordedCall struct {
	tick uint64 //Record the order of all calls relative to each other.
	args []interface{}
}

type spyMethodCall struct {
	*stubbedMethodCall
	recorded []*recordedCall
	subsets  []string
}

func (c *spyMethodCall) calls() []*recordedCall {
	return c.recorded
}
func (c *spyMethodCall) nested() []string {
	return c.subsets
}

func (c *spyMethodCall) String() string {
	//newSliceMatcher[x:y] of
	//	calls after
	//      ">>"
	//		newSliceMatcher[x:y] of
	//  		calls matching(matcher) within
	// 				all calls to <<Other method>>
	//  	"<<"
	//      within
	//     		all calls to <<this method>>
	var rewinds = make([]int, 0)
	depth := 0
	sb := strings.Builder{}
	for i := 0; i < len(c.subsets); i++ {
		if c.subsets[i] == ">>" {
			rewinds = append([]int{depth}, rewinds...)
		} else if c.subsets[i] == "<<" {
			depth = rewinds[0]
			rewinds = rewinds[1:]
		} else {
			if i > 0 {
				sb.WriteRune('\n')
			}
			for d := 0; d < depth; d++ {
				sb.WriteString("  ")
			}
			sb.WriteString(c.subsets[i])
			depth++
		}
	}
	return sb.String()
}

//Setup phase: stub return values
func (c *spyMethodCall) Returning(values ...interface{}) SpyMethodCall {
	c.stubbedMethodCall.Returning(values...)
	return c
}

//Verify phase: expectations on call count
func (c *spyMethodCall) Expect(expect Expectation) {
	count := c.NumCalls()
	if !expect.Met(count) {
		c.t().Errorf("%v expected %v, found %d calls", c, expect, count)
	}
}

func (c *spyMethodCall) Matching(matchers ...interface{}) RecordedCalls {
	matcher := c.receiver.matcher(c.t(), c.m, nil, matchers...)

	var subsetCalls []*recordedCall
	for _, call := range c.recorded {
		if matcher.Matches(call.args...) {
			subsetCalls = append(subsetCalls, call)
		}
	}
	return c.newSubset(subsetCalls, fmt.Sprintf("calls matching %s within", matcher))
}

func (c *spyMethodCall) NumCalls() int {
	return len(c.recorded)
}

func (c *spyMethodCall) Slice(from int, to int) RecordedCalls {
	l := len(c.recorded)
	var subsetCalls []*recordedCall
	var sliceDesc string
	if from < 0 || to < 0 || from > to {
		c.t().Fatalf("Invalid Slice of RecordedCalls %v[%d>:%d]", c, from, to)
	}
	if from > l {
		sliceDesc = fmt.Sprintf("[%d>=len():]", from)
	} else if to > l {
		sliceDesc = fmt.Sprintf("[%d:]", from)
		subsetCalls = c.recorded[from:]
	} else {
		sliceDesc = fmt.Sprintf("[%d:%d]", from, to)
		subsetCalls = c.recorded[from:to]
	}

	return c.newSubset(subsetCalls, fmt.Sprintf("newSliceMatcher%s of", sliceDesc))
}

//Return the calls in c that occurred after those in calls
func (c *spyMethodCall) After(recordedCalls RecordedCalls) RecordedCalls {
	recorded := recordedCalls.calls()

	var subsetCalls []*recordedCall

	if len(recorded) > 0 {
		lastTick := recorded[len(recorded)-1].tick
		if partitionIndex := sort.Search(len(c.recorded), func(i int) bool { return c.recorded[i].tick > lastTick }); partitionIndex < len(c.recorded) {
			subsetCalls = c.recorded[partitionIndex:]
		} // otherwise no matches, default empty set
	} else {
		// all our calls are considered to be after an empty set
		subsetCalls = c.recorded
	}

	nested := append([]string{"calls after", ">>"}, append(recordedCalls.nested(), "<<", "within")...)
	return c.newSubset(subsetCalls, nested...)
}

func newSpyMethodCall(m *method, subsets ...string) *spyMethodCall {

	if len(subsets) == 0 {
		subsets = []string{fmt.Sprintf("all calls to %v", m)}
	}

	call := &spyMethodCall{
		stubbedMethodCall: newStubbedMethodCall(m),
		recorded:          []*recordedCall{},
		subsets:           subsets,
	}
	return call
}

func (c *spyMethodCall) newSubset(calls []*recordedCall, desc ...string) *spyMethodCall {
	subsets := append(desc, c.subsets...)
	result := newSpyMethodCall(c.method, subsets...)
	result.recorded = calls
	return result
}

func (c *spyMethodCall) matches(_ []interface{}) bool {
	return true
}

func (c *spyMethodCall) spy(args []interface{}) ([]interface{}, error) {
	//Spy happens within a method mutex so this is safe..
	c.recorded = append(c.recorded, newRecordedCall(args))
	return c.stubbedMethodCall.spy(args)
}

func newRecordedCall(args []interface{}) *recordedCall {
	return &recordedCall{args: args, tick: atomic.AddUint64(&tick, 1)}
}
