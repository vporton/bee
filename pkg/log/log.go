// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"strconv"
	"sync/atomic"

	"github.com/ethersphere/bee/pkg/log/internal"
)

// TODO: document interfaces!

type builder interface {
	WithName(name string) Logger
	WithValues(keysAndValues ...interface{}) Logger
}

type Verbose interface {
	builder

	Enabled() bool
	V(v uint) Verbose
	Debug(msg string, keysAndValues ...interface{})
}

type Logger interface {
	Verbose

	Info(msg string, keysAndValues ...interface{})

	Warning(msg string, keysAndValues ...interface{})

	Error(err error, msg string, keysAndValues ...interface{})
}

// Marshaler is an optional interface that logged values may choose to
// implement. Loggers with structured output, such as JSON, should
// log the object return by the MarshalLog method instead of the
// original value.
type Marshaler = internal.Marshaler

// Level specifies a level of verbosity for V logs.
// Level should be modified only through its set method.
// Level is treated as a sync/atomic int32.
type Level int32

// get returns the value of the Level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set sets the value of the Level.
func (l *Level) set(val Level) {
	atomic.StoreInt32((*int32)(l), int32(val))
}

// add adds value to this Level and returns the new value.
func (l *Level) add(val Level) Level {
	return Level(atomic.AddInt32((*int32)(l), int32(val.get())))
}

// String implements the fmt.Stringer interface.
func (l *Level) String() string { return strconv.FormatInt(int64(*l), 10) }

const (
	Off = Level(iota - 4)
	Error
	Warning
	Info
	Debug
)

// Logger provides the basic logger functionality.
type basicLogger struct {
	sink      io.Writer
	debugL    uint
	verbosity Level
	formatter internal.Formatter
}

func (l *basicLogger) clone() *basicLogger {
	c := *l
	return &c
}

// Enabled tests whether this Logger is enabled.  For example, commandline
// flags might be used to set the logging verbosity and disable some info logs.
func (l *basicLogger) Enabled() bool {
	return l.verbosity < Off
}

func (l *basicLogger) Debug(msg string, keysAndValues ...interface{}) { // 0
	if int(l.verbosity.get()) >= int(l.debugL) {
		buf := l.formatter.FormatDebug(l.debugL, msg, keysAndValues)
		if _, err := l.sink.Write(buf); err != nil {
			fmt.Printf("log.Debug: unable to write buffer: %s\n", buf)
		}
	}
}

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to the log
// line.  The key/value pairs can then be used to add additional variable
// information.  The key/value pairs must alternate string keys and arbitrary
// values.
func (l *basicLogger) Info(msg string, keysAndValues ...interface{}) { // -1
	if l.verbosity >= Info {
		buf := l.formatter.FormatInfo(msg, keysAndValues)
		if _, err := l.sink.Write(buf); err != nil {
			fmt.Printf("log.Info: unable to write buffer: %s\n", buf)
		}
	}
}

// TODO:
func (l *basicLogger) Warning(msg string, keysAndValues ...interface{}) { // -2
	if l.verbosity >= Warning {
		buf := l.formatter.FormatInfo(msg, keysAndValues)
		if _, err := l.sink.Write(buf); err != nil {
			fmt.Printf("log.Warning: unable to write buffer: %s\n", buf)
		}
	}
	//buf := l.formatter.FormatInfo(msg, keysAndValues)
	//if _, err := l.sink.Write(buf); err != nil {
	//	fmt.Printf("log.Info: unable to write buffer: %s\n", buf)
	//}
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
func (l *basicLogger) Error(err error, msg string, keysAndValues ...interface{}) { // -3
	if l.verbosity >= Error {
		buf := l.formatter.FormatError(err, msg, keysAndValues)
		if _, err := l.sink.Write(buf); err != nil {
			fmt.Printf("log.Error: unable to write buffer: %s\n", buf)
		}
	}
}

// V returns a new Logger instance for a specific verbosity level, relative to
// this Logger.  In other words, V-levels are additive.  A higher verbosity
// level means a log message is less important.  Negative V-levels are treated
// as 0.
func (l *basicLogger) V(level uint) Verbose {
	c := l.clone()
	c.debugL += level
	return c
}

// WithValues returns a new Logger instance with additional key/value pairs.
// See Info for documentation on how key/value pairs work.
func (l *basicLogger) WithValues(keysAndValues ...interface{}) Logger {
	c := l.clone()
	c.formatter.AddValues(keysAndValues)
	return c
}

// WithName returns a new Logger instance with the specified name element added
// to the Logger's name.  Successive calls with WithName append additional
// suffixes to the Logger's name.  It's strongly recommended that name segments
// contain only letters, digits, and hyphens (see the package documentation for
// more information).
func (l *basicLogger) WithName(name string) Logger {
	c := l.clone()
	c.formatter.AddName(name)
	return c
}
