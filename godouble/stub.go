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
)

// StubbedMethodCall is a MethodCall that matches a given set of arguments and returns pre-defined values.
//
type StubbedMethodCall interface {
	/*
			Matching is used to setup whether this call will match a given set of arguments.

		    Empty matcherList list will fatally fail the test

		    If the first matcher is a Matcher then it is used (test will fatally fail is more matcherList are sent)
		    If the first matcher is a func then is equivalent to Matching(Matcher(matcherList[0],matcherList[1:))
		    Otherwise each matcher is converted to a Matcher via either Func() or Eql()
		    and this list is sent to Args()
	*/
	Matching(matchers ...interface{}) StubbedMethodCall

	/*
		Returning is used to setup return values for this call

		The returnValues are converted to a ReturnValues via Values()
	*/
	Returning(returnValues ...interface{}) StubbedMethodCall

	MethodCall
}

type stubbedMethodCall struct {
	*method
	returns ReturnValues
	matcher MethodArgsMatcher
}

func (c *stubbedMethodCall) matches(args []interface{}) bool {
	if c.matcher != nil {
		return c.matcher.Matches(args...)
	}
	return true
}

func (c *stubbedMethodCall) spy(_ []interface{}) ([]interface{}, error) {
	if c.returns == nil {
		c.returns = c.receiver.defaultReturnValues(c.method)
	}
	return c.returns.Receive()
}

func (c *stubbedMethodCall) verify(T) {
	//Nothing to verify
}

func newStubbedMethodCall(m *method) (call *stubbedMethodCall) {
	return &stubbedMethodCall{method: m}
}

func (c *stubbedMethodCall) Returning(returnValues ...interface{}) StubbedMethodCall {
	c.returns = c.receiver.returns(c.t(), c.m, c.returns, returnValues...)
	return c
}

func (c *stubbedMethodCall) Matching(matchers ...interface{}) StubbedMethodCall {
	t := c.method.t()
	t.Helper()
	matcher := c.receiver.matcher(c.t(), c.m, c.matcher, matchers...)
	c.matcher = matcher
	return c
}

func (c *stubbedMethodCall) String() string {
	if c.matcher != nil {
		return fmt.Sprintf("%v matching %v", c.method, c.matcher)
	}
	return c.method.String()
}
