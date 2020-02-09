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

type TDouble struct {
	T
	*TestDouble
}

func NewTDouble(t *testing.T, configs ...func(c *TestDouble)) *TDouble {
	return &TDouble{TestDouble: NewDouble(t, (*T)(nil), configs...)}
}

func (t *TDouble) Errorf(format string, args ...interface{}) {
	t.TestDouble.T().Helper()
	t.Invoke("Errorf", format, args)
}

func (t *TDouble) Fatalf(format string, args ...interface{}) {
	t.TestDouble.T().Helper()
	t.Invoke("Fatalf", format, args)
}

func (t *TDouble) FakeFatalf(format string, args ...interface{}) {
	t.TestDouble.T().Helper()
	panic(fmt.Errorf(format, args...))
}

func (t *TDouble) Logf(format string, args ...interface{}) {
	t.TestDouble.T().Helper()
	t.Invoke("Logf", format, args)
}

func (t *TDouble) Helper() {
	t.TestDouble.T().Helper()
	t.Invoke("Helper")
}

func printfMatcher(re string) Matcher {
	exp := regexp.MustCompile(re)
	f := func(format string, args ...interface{}) bool {
		sprintf := fmt.Sprintf(format, args...)
		return exp.MatchString(sprintf)
	}
	return Func(f, fmt.Sprintf("/%s/", re))
}
