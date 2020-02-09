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

import "fmt"

// An Expectation verifies a count against an expected Value
type Expectation interface {
	// Is the expectation met, complete with count?
	Met(count int) bool
}

//A Completion is an expectation that can indicate that further calls will fail to meet the expectation
//Expectations that are not also Completions are never considered complete
type Completion interface {
	Expectation
	Complete(count int) bool
}

type calledExactly int

func (times calledExactly) Met(count int) bool {
	return count == int(times)
}

func (times calledExactly) Complete(count int) bool {
	return count >= int(times)
}

func (times calledExactly) String() string {
	return fmt.Sprintf("exactly %d", int(times))
}

type calledNever struct{}

func (n *calledNever) Met(count int) bool {
	return count == 0
}
func (n *calledNever) String() string {
	return "never"
}

type calledAtLeast int

func (times calledAtLeast) Met(count int) bool {
	return count >= int(times)
}
func (times calledAtLeast) String() string {
	return fmt.Sprintf("at least %d", int(times))
}

type calledBetween struct {
	atLeast int
	atMost  int
}

func (c calledBetween) Met(count int) bool {
	return count >= c.atLeast && count <= c.atMost
}

func (c calledBetween) Complete(count int) bool {
	return count >= c.atMost
}

func (c calledBetween) String() string {
	if c.atLeast <= 0 {
		return fmt.Sprintf("at most %d", c.atMost)
	}
	return fmt.Sprintf("between %d and %d", c.atLeast, c.atMost)
}

// Exactly returns an expectation to be called exactly n times
// This expectation is considered complete after being exercised n times
func Exactly(n int) Completion {
	return calledExactly(n)
}

// Once is shorthand for Exactly(1)
func Once() Completion {
	return Exactly(1)
}

// Twice is shorthand for Exactly(2)
func Twice() Completion {
	return Exactly(2)
}

var calledNeverSingleton = &calledNever{}

// Never returns an expectation to never be called
func Never() Expectation {
	return calledNeverSingleton
}

// AtLeast returns an expectation to be called at least n times
func AtLeast(n int) Expectation {
	return calledAtLeast(n)
}

// AtMost returns an expectation to be called at most n times
// This expectation is considered complete after being exercised n times
func AtMost(n int) Completion {
	return Between(0, n)
}

// Between returns a new expectation that a method is exercised at least min times and at most max times
// The expectation is considered complete after being exercised max times
func Between(min int, max int) Completion {
	return &calledBetween{min, max}
}
