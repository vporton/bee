// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"os"
	"sync"

	"github.com/ethersphere/bee/pkg/log/internal"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"
}

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

	// modify glog for creating local logger instances
	// modify loggr.functor for not having levels...
	// INFO, WARN, ERROR - normal
	// DEBUG - with V levels
	// Flat hierarchy with tree-emulated hierarchy using string: "root", "root/child1", etc...

	logger = &basicLogger{
		sink: os.Stderr,
		formatter: internal.NewFormatter(
			internal.Options{
				LogTimestamp: true,
				LogCaller:    internal.All,
				//LogCallerFunc: true,
			},
		),
		verbosity: 2,
	}
	registry.loggers[name] = logger
	return logger
}
