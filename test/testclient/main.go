// Copyright 2017 Google LLC
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
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

var addr = flag.String("addr", "", "Server address")
var numWrites = flag.Int("w", 5, "number of threads for writing")
var numReads = flag.Int("r", 10, "number of threads for reading")
var keySpace = flag.Int("key-space", 100000, "size of the key space")
var dataSize = flag.Int("data-size", 1000, "size of data to write (in bytes)")

type Stats struct {
	CacheHits   int64
	CacheMisses int64
	Writes      int64
	Errors      int64
	Lock        sync.Mutex
}

var keyMap = make(map[int]string)

func main() {
	flag.Parse()

	for i := 0; i < *keySpace; i++ {
		hasher := sha1.New()
		fmt.Fprintf(hasher, "%d", i)
		keyMap[i] = hex.EncodeToString(hasher.Sum(nil))
	}

	done := make(chan struct{})
	// Watch for SIGINT or SIGTERM and cancel
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		s := <-signals
		fmt.Printf("Received signal %s, exiting...", s)
		done <- struct{}{}
	}()

	mc := memcache.New(*addr)

	reads := make(chan bool, 1000)
	writes := make(chan struct{}, 1000)
	errors := make(chan error, 1000)

	stats := &Stats{}
	go printer(stats, done)
	go counter(stats, reads, writes, errors, done)
	for i := 0; i < *numReads; i++ {
		go readLoop(mc, reads, errors, done)
	}
	for i := 0; i < *numWrites; i++ {
		go writeLoop(mc, writes, errors, done)
	}

	// block and wait for the done channel
	<-done
}

func printer(stats *Stats, done <-chan struct{}) {
	for {
		select {
		case <-time.After(5 * time.Second):
			stats.Lock.Lock()
			fmt.Printf("Hits: %d, Misses: %d, Writes: %d, Errors: %d\n", stats.CacheHits, stats.CacheMisses, stats.Writes, stats.Errors)
			stats.CacheHits = 0
			stats.CacheMisses = 0
			stats.Writes = 0
			stats.Errors = 0
			stats.Lock.Unlock()
		case <-done:
			return
		}
	}
}

func counter(stats *Stats, r <-chan bool, w <-chan struct{}, e <-chan error, done <-chan struct{}) {
	for {
		select {
		case h := <-r:
			stats.Lock.Lock()
			if h {
				stats.CacheHits += 1
			} else {
				stats.CacheMisses += 1
			}
			stats.Lock.Unlock()
		case <-w:
			stats.Lock.Lock()
			stats.Writes += 1
			stats.Lock.Unlock()
		case <-e:
			stats.Lock.Lock()
			stats.Errors += 1
			stats.Lock.Unlock()
		case <-done:
			return
		}
	}
}

// A key generator that generates a random key from the key space.
func getKey() string {
	intval := rand.Intn(*keySpace)
	return keyMap[intval]
}

func readLoop(mc *memcache.Client, r chan<- bool, e chan<- error, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			key := getKey()
			_, err := mc.Get(key)
			if err == nil {
				r <- true
			} else {
				if err == memcache.ErrCacheMiss {
					r <- false
				} else {
					e <- err
				}
			}
		}
	}
}

func writeLoop(mc *memcache.Client, w chan<- struct{}, e chan<- error, done <-chan struct{}) {
	// Just write the same thing each time for this thread
	data := make([]byte, *dataSize)
	rand.Read(data)

	for {
		select {
		case <-done:
			return
		default:
			key := getKey()
			err := mc.Set(&memcache.Item{
				Key:   key,
				Value: data,
			})
			w <- struct{}{}
			if err != nil {
				e <- err
			}
		}
	}
}
