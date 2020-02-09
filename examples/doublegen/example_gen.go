// +build doublegen
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

package main

import (
	"github.com/lwoggardner/godouble/doublegen"
	"github.com/lwoggardner/godouble/examples"
	"log"
	"os"
)

func main() {
	if f, e := os.Create("example_double_test.go"); e == nil {
		defer f.Close()
		doublegen.NewGenerator((*examples.API)(nil)).GenerateDouble(f)
	} else {
		log.Fatal(e)
	}
}
