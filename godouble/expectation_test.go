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
	"regexp"
	"testing"
)

func TestExpectations(t *testing.T) {
	type test struct {
		name string
		Expectation
		truthy     []int
		falsey     []int
		complete   []int
		incomplete []int
		re         string
	}

	tests := []test{
		{"Between", Between(5, 11), []int{5, 11, 7}, []int{4, 0, -1, 12, 65434}, []int{11, 12, 1254}, []int{0, 5, 10}, "between 5 and 11"},
		{"Exactly", Exactly(5), []int{5}, []int{4, 0, -1, 12, 65434}, []int{5, 6, 1235}, []int{0, 4}, "exactly 5"},
		{"Never", Never(), []int{0}, []int{-1, 1, 1124}, nil, []int{0, 1, 123}, "never"},
		{"AtLeast", AtLeast(6), []int{6, 7, 125}, []int{-1, 1, 5}, nil, []int{0, 1, 5, 6, 7, 123}, "at least 6"},
		{"AtMost", AtMost(10), []int{10, 9, 0, 1}, []int{11, 1124}, []int{10, 11, 124}, []int{0, 1, 9}, "at most 10"},
	}

	for _, tt := range tests {
		ex := tt
		t.Run(ex.name, func(t *testing.T) {
			if !regexp.MustCompile(ex.re).MatchString(fmt.Sprint(ex.Expectation)) {
				t.Errorf("Expected %v to match /%s/", ex.Expectation, ex.re)
			}
			for _, expectTrue := range ex.truthy {
				if !ex.Met(expectTrue) {
					t.Errorf("expected %v to be met for %d, but is not", ex, expectTrue)
				}
			}

			for _, expectFalse := range ex.falsey {
				if ex.Met(expectFalse) {
					t.Errorf("Expected %v to not be met for %d, but is", ex, expectFalse)
				}
			}

			if completion, isCompletion := ex.Expectation.(Completion); isCompletion {
				for _, expectComplete := range ex.complete {
					if !completion.Complete(expectComplete) {
						t.Errorf("Expected %v to be complete for %d, but is not", ex, expectComplete)
					}
				}
				for _, expectIncomplete := range ex.incomplete {
					if completion.Complete(expectIncomplete) {
						t.Errorf("Expected %v to not be complete for %d, but is", ex, expectIncomplete)
					}
				}

			} else if len(ex.complete) > 0 {
				t.Errorf("Expected %v to be incomplete for %v, but it is not a Completion", ex, ex.complete)
			}

		})
	}
}
