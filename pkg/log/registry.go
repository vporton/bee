// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"os"
	"sync"
)

// registry is the central register for Logger instances.
var registry = struct {
	sync.RWMutex

	levels  map[string]int
	loggers map[string]*basicLogger
}{
	levels:  make(map[string]int),
	loggers: make(map[string]*basicLogger),
}

func NewLogger(name string) *basicLogger {
	registry.Lock()
	defer registry.Unlock()

	logger, ok := registry.loggers[name]
	if ok {
		return logger
	}

	// TODO:
	// - Cleanup log.go and formatter.go files; with all todos
	// - Establish global vars for modifying instances behaviour on creation (guard them with sync.Once)
	// - Implement the global log registry with ability to change verbosity of separate loggers
	// - Write example file
	// - Write short demonstration program with several goroutines
	// - Add V-level tests
	// - Write doc.go
	// - Write benchmarks
	// - Do optimizations

	// Flat hierarchy with tree-emulated hierarchy using string: "root", "root/child1", etc...

	logger = newBasicLogger(
		newFormatter(
			fmtOptions{
				logTimestamp: true,
				logCaller:    categoryAll,
				//logCallerFunc: true,
			},
		),
		os.Stderr,
		VerbosityAll,
	)
	registry.loggers[name] = logger
	return logger
}
