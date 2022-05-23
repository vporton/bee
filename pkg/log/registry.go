// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"sync"

	"github.com/ethersphere/bee/pkg/log/internal"
	"github.com/go-logr/zapr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"
}

// registry is the central register for Logger instances.
var registry = struct {
	sync.RWMutex

	levels  map[string]zap.AtomicLevel
	loggers map[string]*zap.Logger
}{
	levels:  make(map[string]zap.AtomicLevel),
	loggers: make(map[string]*zap.Logger),
}

func NewLogger(name string) *Logger {
	registry.Lock()
	defer registry.Unlock()

	logger, ok := registry.loggers[name]
	if ok {
		return &Logger{zapr.NewLogger(logger)}
	}

	logger := internal.VLogger{}

	registry.loggers[name] = logger
	return &Logger{zapr.NewLogger(logger)}
}
