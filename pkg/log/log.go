// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"github.com/ethersphere/bee/pkg/log/internal"
	"github.com/go-logr/logr"
)

// Marshaler is an optional interface that logged values may choose to
// implement. Loggers with structured output, such as JSON, should
// log the object return by the MarshalLog method instead of the
// original value.
type Marshaler = internal.Marshaler

// Logger provides the basic logger functionality.
type Logger struct {
	logger logr.Logger
}

// Enabled tests whether this Logger is enabled.  For example, commandline
// flags might be used to set the logging verbosity and disable some info logs.
func (l Logger) Enabled() bool {
	return l.logger.Enabled()
}

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to the log
// line.  The key/value pairs can then be used to add additional variable
// information.  The key/value pairs must alternate string keys and arbitrary
// values.
func (l Logger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as context.
// It functions similarly to Info, but may have unique behavior, and should be
// preferred for logging errors (see the package documentations for more
// information). The log message will always be emitted, regardless of
// verbosity level.
//
// The msg argument should be used to add context to any underlying error,
// while the err argument should be used to attach the actual error that
// triggered this log line, if present. The err parameter is optional
// and nil may be passed instead of an error instance.
func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.logger.Error(err, msg, keysAndValues...)
}

// V returns a new Logger instance for a specific verbosity level, relative to
// this Logger.  In other words, V-levels are additive.  A higher verbosity
// level means a log message is less important.  Negative V-levels are treated
// as 0.
func (l Logger) V(level int) Logger {
	return Logger{logger: l.logger.V(level)}
}

// WithValues returns a new Logger instance with additional key/value pairs.
// See Info for documentation on how key/value pairs work.
func (l Logger) WithValues(keysAndValues ...interface{}) Logger {
	return Logger{logger: l.logger.WithValues(keysAndValues...)}
}

// WithName returns a new Logger instance with the specified name element added
// to the Logger's name.  Successive calls with WithName append additional
// suffixes to the Logger's name.  It's strongly recommended that name segments
// contain only letters, digits, and hyphens (see the package documentation for
// more information).
func (l Logger) WithName(name string) Logger {
	return Logger{logger: l.logger.WithName(name)}
}

// WithCallDepth returns a Logger instance that offsets the call stack by the
// specified number of frames when logging call site information, if possible.
// This is useful for users who have helper functions between the "real" call
// site and the actual calls to Logger methods.  If depth is 0 the attribution
// should be to the direct caller of this function.  If depth is 1 the
// attribution should skip 1 call frame, and so on.  Successive calls to this
// are additive.
//
// If the underlying log implementation supports a WithCallDepth(int) method,
// it will be called and the result returned.  If the implementation does not
// support CallDepthLogSink, the original Logger will be returned.
//
// To skip one level, WithCallStackHelper() should be used instead of
// WithCallDepth(1) because it works with implementions that support the
// CallDepthLogSink and/or CallStackHelperLogSink interfaces.
func (l Logger) WithCallDepth(depth int) Logger {
	return Logger{logger: l.logger.WithCallDepth(depth)}
}

// WithCallStackHelper returns a new Logger instance that skips the direct
// caller when logging call site information, if possible.  This is useful for
// users who have helper functions between the "real" call site and the actual
// calls to Logger methods and want to support loggers which depend on marking
// each individual helper function, like loggers based on testing.T.
//
// In addition to using that new logger instance, callers also must call the
// returned function.
//
// If the underlying log implementation supports a WithCallDepth(int) method,
// WithCallDepth(1) will be called to produce a new logger. If it supports a
// WithCallStackHelper() method, that will be also called. If the
// implementation does not support either of these, the original Logger will be
// returned.
func (l Logger) WithCallStackHelper() (func(), Logger) {
	fn, lgr := l.logger.WithCallStackHelper()
	return fn, Logger{logger: lgr}
}
