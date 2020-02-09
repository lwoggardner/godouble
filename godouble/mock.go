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

//MockedMethodCall is a MethodCall that has pre-defined expectations for how often and sequence of invocations
type MockedMethodCall interface {
	/*
			Matching is used to setup whether this call will match a given set of arguments.

		    Empty matcherList list will fatally fail the test

		    If the first matcher is a Matcher then it is used (test will fatally fail is more matcherList are sent)
		    If the first matcher is a func then is equivalent to Matching(Matcher(matcherList[0],matcherList[1:))
		    Otherwise each matcher is converted to a Matcher via either Func() or Eql()
		    and this list is sent to Args()
	*/
	Matching(matchers ...interface{}) MockedMethodCall

	//Setup that this call will only match if the supplied calls are already complete
	After(calls ...MockedMethodCall) MockedMethodCall

	/*
		Returning is used to setup return values for this call

		The returnValues are converted to a ReturnValues via Values()
	*/
	Returning(values ...interface{}) MockedMethodCall

	//Setup an expectation on the number of times this call will be invoked
	Expect(expect Expectation) MockedMethodCall

	MethodCall

	complete() bool
}

type mockedMethodCall struct {
	*stubbedMethodCall
	count  int
	after  []MockedMethodCall
	expect Expectation
}

func (c *mockedMethodCall) complete() bool {
	if completion, isCompletion := c.expect.(Completion); isCompletion {
		return completion.Complete(c.count)
	}
	return false
}

func (c *mockedMethodCall) met() bool {
	if c.expect != nil {
		return c.expect.Met(c.count)
	}
	return true
}

func newMockedMethodCall(m *method) MockedMethodCall {

	call := &mockedMethodCall{
		stubbedMethodCall: newStubbedMethodCall(m),
		count:             0,
		after:             []MockedMethodCall{},
	}
	return call
}

func (c *mockedMethodCall) Matching(matchers ...interface{}) MockedMethodCall {
	c.t().Helper()
	c.stubbedMethodCall.Matching(matchers...)
	return c
}

//This stubbedMethodCall will only be invoked after these other methods (which might be on other mocks) have been met
func (c *mockedMethodCall) After(after ...MockedMethodCall) MockedMethodCall {
	c.after = append(c.after, after...)
	return c
}

func (c *mockedMethodCall) Returning(values ...interface{}) MockedMethodCall {
	c.stubbedMethodCall.Returning(values...)
	return c
}

func (c *mockedMethodCall) Expect(expect Expectation) MockedMethodCall {
	c.expect = expect
	return c
}

func (c *mockedMethodCall) inSequence() bool {
	for _, call := range c.after {
		if !call.complete() {
			return false
		}
	}
	return true
}

func (c *mockedMethodCall) matches(args []interface{}) bool {
	return c.stubbedMethodCall.matches(args) && !c.complete() && c.inSequence()
}

func (c *mockedMethodCall) spy(args []interface{}) ([]interface{}, error) {
	c.count++
	if c.trace() && c.complete() {
		c.t().Logf("%v completed expectations after %d calls", c, c.count)
	}
	return c.stubbedMethodCall.spy(args)
}

func (c *mockedMethodCall) verify(t T) {
	t.Helper()
	if !c.met() {
		t.Errorf("%v expected %v, found %d calls", c.stubbedMethodCall, c.expect, c.count)
	}
}

// ExpectInOrder is shorthand to Setup that the list of calls are expected to executed in this sequence
func ExpectInOrder(calls ...MockedMethodCall) {
	for i := len(calls) - 1; i > 0; i-- {
		calls[i].After(calls[i-1])
	}
}
