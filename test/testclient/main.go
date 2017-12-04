// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
)

var addr = flag.String("addr", "", "Server address")
var write = flag.Bool("write", false, "Write the key/data to memcached")
var key = flag.String("key", "foo", "Memcached key to read/write")
var data = flag.String("data", "", "Memcached data to write")

func main() {
	flag.Parse()

	mc := memcache.New(*addr)
	if *write {
		err := mc.Set(&memcache.Item{
			Key:   *key,
			Value: []byte(*data),
		})
		if err != nil {
			fmt.Printf("Error writing to memcached: %v", err)
		}
		return
	}

	it, err := mc.Get(*key)
	if err != nil {
		fmt.Printf("Error reading from memcached: %v", err)
		return
	}

	fmt.Printf("%s", it.Value)
}
