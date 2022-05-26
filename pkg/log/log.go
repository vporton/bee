// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
)

// TODO: document interfaces!
// TODO: use generics to alternate type on builder methods from Logger to Verbose to allow only the relevant methods on: logger.V(1).WithName("some_name").
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

// Level specifies a level of verbosity for logger.
// Level should be modified only through its set method.
// Level is treated as a sync/atomic int32.
type Level int32

// get returns the value of the Level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set updates the value of the Level.
func (l *Level) set(v Level) {
	atomic.StoreInt32((*int32)(l), int32(v))
}

// String implements the fmt.Stringer interface.
func (l *Level) String() string { return strconv.FormatInt(int64(*l), 10) }

const (
	VerbosityAll  = Level(1<<31 - 1)
	VerbosityNone = Level(iota - 4)
	VerbosityError
	VerbosityWarning
	VerbosityInfo
	VerbosityDebug
)

// Lock wraps io.Writer in a mutex to make it safe for concurrent use.
// In particular, *os.Files must be locked before use.
func Lock(w io.Writer) io.Writer {
	if _, ok := w.(*lockWriter); ok {
		return w // No need to layer on another lock.
	}
	return &lockWriter{w: w}
}

type lockWriter struct {
	sync.Mutex
	w io.Writer
}

func (ls *lockWriter) Write(bs []byte) (int, error) {
	ls.Lock()
	n, err := ls.w.Write(bs)
	ls.Unlock()
	return n, err
}

// newBasicLogger is a convenient constructor for basicLogger.
func newBasicLogger(fmt Formatter, sink io.Writer, verbosity Level) *basicLogger {
	return &basicLogger{
		fmt:       fmt,
		sink:      sink,
		verbosity: verbosity,
	}
}

// basicLogger provides the basic logger functionality.
type basicLogger struct {
	// TODO: protect sink.Write with sync.Mutex !?

	// fmt formats logger messages before they are written to the sink.
	fmt Formatter // TODO: share formatter by pointer.
	// sink represents the stream where the logs are written.
	sink io.Writer
	// debugL represents the verbosity V level for the debug calls.
	// Higher values enable more logs. Debug logs at or below this level
	// will be written, while logs above this level will be discarded.
	debugL uint
	// verbosity represents the current verbosity level.
	// This variable is used to change the verbosity of the logger instance.
	verbosity Level
}

// clone returns a clone the basicLogger.
func (l *basicLogger) clone() *basicLogger {
	// TODO: check in registry if such logger exists (guard with lock)?
	c := *l
	return &c
}

// setVerbosity changes the verbosity level or the logger.
func (l *basicLogger) setVerbosity(v Level) {
	l.verbosity.set(v)
}

// Enabled tests whether this Logger is enabled.
func (l *basicLogger) Enabled() bool {
	return l.verbosity.get() > VerbosityNone
}

func (l *basicLogger) Debug(msg string, keysAndValues ...interface{}) { // 0
	if int(l.verbosity.get()) >= int(l.debugL) {
		args := l.fmt.base("debug")
		if l.debugL > 0 {
			args = append(args, "v", l.debugL)
		}
		if policy := l.fmt.opts.logCaller; policy == categoryAll || policy == categoryDebug {
			args = append(args, "caller", l.fmt.caller())
		}
		args = append(args, "msg", msg)
		buf := l.fmt.render(args, keysAndValues)
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
	if l.verbosity.get() >= VerbosityInfo {
		args := l.fmt.base("info")
		if policy := l.fmt.opts.logCaller; policy == categoryAll || policy == categoryInfo {
			args = append(args, "caller", l.fmt.caller())
		}
		args = append(args, "msg", msg)
		buf := l.fmt.render(args, keysAndValues)
		if _, err := l.sink.Write(buf); err != nil {
			fmt.Printf("log.Info: unable to write buffer: %s\n", buf)
		}
	}
}

func (l *basicLogger) Warning(msg string, keysAndValues ...interface{}) { // -2
	if l.verbosity.get() >= VerbosityWarning {
		args := l.fmt.base("warning")
		if policy := l.fmt.opts.logCaller; policy == categoryAll || policy == categoryWarning {
			args = append(args, "caller", l.fmt.caller())
		}
		args = append(args, "msg", msg)
		buf := l.fmt.render(args, keysAndValues)
		if _, err := l.sink.Write(buf); err != nil {
			fmt.Printf("log.Warning: unable to write buffer: %s\n", buf)
		}
	}
	//buf := l.fmt.FormatInfo(msg, keysAndValues)
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
func (l *basicLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if l.verbosity.get() >= VerbosityError {
		args := l.fmt.base("error")
		if policy := l.fmt.opts.logCaller; policy == categoryAll || policy == categoryError {
			args = append(args, "caller", l.fmt.caller())
		}
		args = append(args, "msg", msg)
		var loggableErr interface{}
		if err != nil {
			loggableErr = err.Error()
		}
		args = append(args, "error", loggableErr)
		buf := l.fmt.render(args, keysAndValues)
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
	c.fmt.AddValues(keysAndValues)
	return c
}

// WithName returns a new Logger instance with the specified name element added
// to the Logger's name.  Successive calls with WithName append additional
// suffixes to the Logger's name.  It's strongly recommended that name segments
// contain only letters, digits, and hyphens (see the package documentation for
// more information).
func (l *basicLogger) WithName(name string) Logger {
	c := l.clone()
	c.fmt.AddName(name)
	return c
}
