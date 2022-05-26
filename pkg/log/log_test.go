// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Note: the following code is derived (borrows) from: github.com/go-logr/logr

package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"testing"
)

// applyError is a higher order function that returns fn with an applied err.
func applyError(fn func(error, string, ...interface{}), err error) func(string, ...interface{}) {
	return func(msg string, kvs ...interface{}) {
		fn(err, msg, kvs...)
	}
}

func TestLoggerOptionsTimestampFormat(t *testing.T) {
	bb := new(bytes.Buffer)
	logger := newBasicLogger(newFormatter(fmtOptions{logTimestamp: true, timestampFormat: "TIMESTAMP", callerDepth: 1}), bb, VerbosityAll)
	logger.Info("msg")
	have := string(bytes.TrimRight(bb.Bytes(), "\n"))
	want := `"time"="TIMESTAMP" "level"="info" "logger"="root" "msg"="msg"`
	if have != want {
		t.Errorf("\nwant %q\nhave %q", want, have)
	}
}

func TestLoggerEnabled(t *testing.T) {
	t.Run("verbosity none|all", func(t *testing.T) {
		log := newBasicLogger(newFormatter(fmtOptions{}), io.Discard, VerbosityNone)
		if log.Enabled() {
			t.Errorf("want false")
		}
		log.setVerbosity(VerbosityAll)
		if !log.Enabled() {
			t.Errorf("want true")
		}
	})
}

func TestLogger(t *testing.T) {
	bb := new(bytes.Buffer)
	logger := newBasicLogger(newFormatter(fmtOptions{}), bb, VerbosityAll)

	testCases := []struct {
		name  string
		logFn func(string, ...interface{})
		args  []interface{}
		want  string
	}{{
		name:  "just msg",
		logFn: logger.Debug,
		args:  makeKV(),
		want:  `"level"="debug" "logger"="root" "msg"="msg"`,
	}, {
		name:  "primitives",
		logFn: logger.Debug,
		args:  makeKV("int", 1, "str", "ABC", "bool", true),
		want:  `"level"="debug" "logger"="root" "msg"="msg" "int"=1 "str"="ABC" "bool"=true`,
	}, {
		name:  "just msg",
		logFn: logger.Info,
		args:  makeKV(),
		want:  `"level"="info" "logger"="root" "msg"="msg"`,
	}, {
		name:  "primitives",
		logFn: logger.Info,
		args:  makeKV("int", 1, "str", "ABC", "bool", true),
		want:  `"level"="info" "logger"="root" "msg"="msg" "int"=1 "str"="ABC" "bool"=true`,
	}, {
		name:  "just msg",
		logFn: logger.Warning,
		args:  makeKV(),
		want:  `"level"="warning" "logger"="root" "msg"="msg"`,
	}, {
		name:  "primitives",
		logFn: logger.Warning,
		args:  makeKV("int", 1, "str", "ABC", "bool", true),
		want:  `"level"="warning" "logger"="root" "msg"="msg" "int"=1 "str"="ABC" "bool"=true`,
	}, {
		name:  "just msg",
		logFn: applyError(logger.Error, errors.New("err")),
		args:  makeKV(),
		want:  `"level"="error" "logger"="root" "msg"="msg" "error"="err"`,
	}, {
		name:  "primitives",
		logFn: applyError(logger.Error, errors.New("err")),
		args:  makeKV("int", 1, "str", "ABC", "bool", true),
		want:  `"level"="error" "logger"="root" "msg"="msg" "error"="err" "int"=1 "str"="ABC" "bool"=true`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bb.Reset()
			tc.logFn("msg", tc.args...)
			have := string(bytes.TrimRight(bb.Bytes(), "\n"))
			if have != tc.want {
				t.Errorf("\nwant %q\nhave %q", tc.want, have)
			}
		})
	}
}

func TestLoggerWithCaller(t *testing.T) {
	t.Run("logCaller=categoryAll", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryAll}), bb, VerbosityAll)

		logger.Debug("msg")
		_, file, line, _ := runtime.Caller(0)
		want := fmt.Sprintf(`"level"="debug" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line-1)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		_, file, line, _ = runtime.Caller(0)
		want = fmt.Sprintf(`"level"="info" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line-1)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		_, file, line, _ = runtime.Caller(0)
		want = fmt.Sprintf(`"level"="warning" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line-1)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		_, file, line, _ = runtime.Caller(0)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = fmt.Sprintf(`"level"="error" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg" "error"="err"`, filepath.Base(file), line-1)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("logCaller=categoryAll, logCallerFunc=true", func(t *testing.T) {
		const thisFunc = "github.com/ethersphere/bee/pkg/log.TestLoggerWithCaller.func2"

		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryAll, logCallerFunc: true}), bb, VerbosityAll)

		logger.Debug("msg")
		_, file, line, _ := runtime.Caller(0)
		want := fmt.Sprintf(`"level"="debug" "logger"="root" "caller"={"file":%q,"line":%d,"function":%q} "msg"="msg"`, filepath.Base(file), line-1, thisFunc)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		_, file, line, _ = runtime.Caller(0)
		want = fmt.Sprintf(`"level"="info" "logger"="root" "caller"={"file":%q,"line":%d,"function":%q} "msg"="msg"`, filepath.Base(file), line-1, thisFunc)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		_, file, line, _ = runtime.Caller(0)
		want = fmt.Sprintf(`"level"="warning" "logger"="root" "caller"={"file":%q,"line":%d,"function":%q} "msg"="msg"`, filepath.Base(file), line-1, thisFunc)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		_, file, line, _ = runtime.Caller(0)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = fmt.Sprintf(`"level"="error" "logger"="root" "caller"={"file":%q,"line":%d,"function":%q} "msg"="msg" "error"="err"`, filepath.Base(file), line-1, thisFunc)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("logCaller=categoryDebug", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryDebug}), bb, VerbosityAll)

		logger.Debug("msg")
		_, file, line, _ := runtime.Caller(0)
		want := fmt.Sprintf(`"level"="debug" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line-1)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		want = `"level"="info" "logger"="root" "msg"="msg"`
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		want = `"level"="warning" "logger"="root" "msg"="msg"`
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="error" "logger"="root" "msg"="msg" "error"="err"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("logCaller=categoryInfo", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryInfo}), bb, VerbosityAll)

		logger.Debug("msg")
		want := `"level"="debug" "logger"="root" "msg"="msg"`
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		_, file, line, _ := runtime.Caller(0)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = fmt.Sprintf(`"level"="info" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line-1)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="warning" "logger"="root" "msg"="msg"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="error" "logger"="root" "msg"="msg" "error"="err"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("logCaller=categoryWarning", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryWarning}), bb, VerbosityAll)

		logger.Debug("msg")
		want := `"level"="debug" "logger"="root" "msg"="msg"`
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="info" "logger"="root" "msg"="msg"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		_, file, line, _ := runtime.Caller(0)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = fmt.Sprintf(`"level"="warning" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line-1)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="error" "logger"="root" "msg"="msg" "error"="err"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("logCaller=categoryError", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryError}), bb, VerbosityAll)

		logger.Debug("msg")
		want := `"level"="debug" "logger"="root" "msg"="msg"`
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="info" "logger"="root" "msg"="msg"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="warning" "logger"="root" "msg"="msg"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		_, file, line, _ := runtime.Caller(0)
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = fmt.Sprintf(`"level"="error" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg" "error"="err"`, filepath.Base(file), line-1)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("logCaller=categoryNone", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryNone}), bb, VerbosityAll)

		logger.Debug("msg")
		want := `"level"="debug" "logger"="root" "msg"="msg"`
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Info("msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="info" "logger"="root" "msg"="msg"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Warning("msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="warning" "logger"="root" "msg"="msg"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}

		bb.Reset()

		logger.Error(errors.New("err"), "msg")
		have = string(bytes.TrimRight(bb.Bytes(), "\n"))
		want = `"level"="error" "logger"="root" "msg"="msg" "error"="err"`
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
}

func TestLoggerWithName(t *testing.T) {
	bb := new(bytes.Buffer)
	logger := newBasicLogger(newFormatter(fmtOptions{}), bb, VerbosityAll)

	testCases := []struct {
		name  string
		logFn func(string, ...interface{})
		args  []interface{}
		want  string
	}{{
		name:  "one",
		logFn: logger.WithName("pfx1").Debug,
		args:  makeKV("k", "v"),
		want:  `"level"="debug" "logger"="root/pfx1" "msg"="msg" "k"="v"`,
	}, {
		name:  "two",
		logFn: logger.WithName("pfx1").WithName("pfx2").Debug,
		args:  makeKV("k", "v"),
		want:  `"level"="debug" "logger"="root/pfx1/pfx2" "msg"="msg" "k"="v"`,
	}, {
		name:  "one",
		logFn: logger.WithName("pfx1").Info,
		args:  makeKV("k", "v"),
		want:  `"level"="info" "logger"="root/pfx1" "msg"="msg" "k"="v"`,
	}, {
		name:  "two",
		logFn: logger.WithName("pfx1").WithName("pfx2").Info,
		args:  makeKV("k", "v"),
		want:  `"level"="info" "logger"="root/pfx1/pfx2" "msg"="msg" "k"="v"`,
	}, {
		name:  "one",
		logFn: logger.WithName("pfx1").Warning,
		args:  makeKV("k", "v"),
		want:  `"level"="warning" "logger"="root/pfx1" "msg"="msg" "k"="v"`,
	}, {
		name:  "two",
		logFn: logger.WithName("pfx1").WithName("pfx2").Warning,
		args:  makeKV("k", "v"),
		want:  `"level"="warning" "logger"="root/pfx1/pfx2" "msg"="msg" "k"="v"`,
	}, {
		name:  "one",
		logFn: applyError(logger.WithName("pfx1").Error, errors.New("err")),
		args:  makeKV("k", "v"),
		want:  `"level"="error" "logger"="root/pfx1" "msg"="msg" "error"="err" "k"="v"`,
	}, {
		name:  "two",
		logFn: applyError(logger.WithName("pfx1").WithName("pfx2").Error, errors.New("err")),
		args:  makeKV("k", "v"),
		want:  `"level"="error" "logger"="root/pfx1/pfx2" "msg"="msg" "error"="err" "k"="v"`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bb.Reset()
			tc.logFn("msg", tc.args...)
			have := string(bytes.TrimRight(bb.Bytes(), "\n"))
			if have != tc.want {
				t.Errorf("\nwant %q\nhave %q", tc.want, have)
			}
		})
	}
}

func TestLoggerWithValues(t *testing.T) {
	bb := new(bytes.Buffer)
	logger := newBasicLogger(newFormatter(fmtOptions{}), bb, VerbosityAll)

	testCases := []struct {
		name  string
		logFn func(string, ...interface{})
		args  []interface{}
		want  string
	}{{
		name:  "zero",
		logFn: logger.Debug,
		args:  makeKV("k", "v"),
		want:  `"level"="debug" "logger"="root" "msg"="msg" "k"="v"`,
	}, {
		name:  "one",
		logFn: logger.WithValues("one", 1).Debug,
		args:  makeKV("k", "v"),
		want:  `"level"="debug" "logger"="root" "msg"="msg" "one"=1 "k"="v"`,
	}, {
		name:  "two",
		logFn: logger.WithValues("one", 1, "two", 2).Debug,
		args:  makeKV("k", "v"),
		want:  `"level"="debug" "logger"="root" "msg"="msg" "one"=1 "two"=2 "k"="v"`,
	}, {
		name:  "dangling",
		logFn: logger.WithValues("dangling").Debug,
		args:  makeKV("k", "v"),
		want:  `"level"="debug" "logger"="root" "msg"="msg" "dangling"="<no-value>" "k"="v"`,
	}, {
		name:  "zero",
		logFn: logger.Info,
		args:  makeKV("k", "v"),
		want:  `"level"="info" "logger"="root" "msg"="msg" "k"="v"`,
	}, {
		name:  "one",
		logFn: logger.WithValues("one", 1).Info,
		args:  makeKV("k", "v"),
		want:  `"level"="info" "logger"="root" "msg"="msg" "one"=1 "k"="v"`,
	}, {
		name:  "two",
		logFn: logger.WithValues("one", 1, "two", 2).Info,
		args:  makeKV("k", "v"),
		want:  `"level"="info" "logger"="root" "msg"="msg" "one"=1 "two"=2 "k"="v"`,
	}, {
		name:  "dangling",
		logFn: logger.WithValues("dangling").Info,
		args:  makeKV("k", "v"),
		want:  `"level"="info" "logger"="root" "msg"="msg" "dangling"="<no-value>" "k"="v"`,
	}, {
		name:  "zero",
		logFn: logger.Warning,
		args:  makeKV("k", "v"),
		want:  `"level"="warning" "logger"="root" "msg"="msg" "k"="v"`,
	}, {
		name:  "one",
		logFn: logger.WithValues("one", 1).Warning,
		args:  makeKV("k", "v"),
		want:  `"level"="warning" "logger"="root" "msg"="msg" "one"=1 "k"="v"`,
	}, {
		name:  "two",
		logFn: logger.WithValues("one", 1, "two", 2).Warning,
		args:  makeKV("k", "v"),
		want:  `"level"="warning" "logger"="root" "msg"="msg" "one"=1 "two"=2 "k"="v"`,
	}, {
		name:  "dangling",
		logFn: logger.WithValues("dangling").Warning,
		args:  makeKV("k", "v"),
		want:  `"level"="warning" "logger"="root" "msg"="msg" "dangling"="<no-value>" "k"="v"`,
	}, {
		name:  "zero",
		logFn: applyError(logger.Error, errors.New("err")),
		args:  makeKV("k", "v"),
		want:  `"level"="error" "logger"="root" "msg"="msg" "error"="err" "k"="v"`,
	}, {
		name:  "one",
		logFn: applyError(logger.WithValues("one", 1).Error, errors.New("err")),
		args:  makeKV("k", "v"),
		want:  `"level"="error" "logger"="root" "msg"="msg" "error"="err" "one"=1 "k"="v"`,
	}, {
		name:  "two",
		logFn: applyError(logger.WithValues("one", 1, "two", 2).Error, errors.New("err")),
		args:  makeKV("k", "v"),
		want:  `"level"="error" "logger"="root" "msg"="msg" "error"="err" "one"=1 "two"=2 "k"="v"`,
	}, {
		name:  "dangling",
		logFn: applyError(logger.WithValues("dangling").Error, errors.New("err")),
		args:  makeKV("k", "v"),
		want:  `"level"="error" "logger"="root" "msg"="msg" "error"="err" "dangling"="<no-value>" "k"="v"`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bb.Reset()
			tc.logFn("msg", tc.args...)
			have := string(bytes.TrimRight(bb.Bytes(), "\n"))
			if have != tc.want {
				t.Errorf("\nwant %q\nhave %q", tc.want, have)
			}
		})
	}
}

func TestLoggerWithCallDepth(t *testing.T) {
	t.Run("level=debug, callerDepth=1", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryAll, callerDepth: 1}), bb, VerbosityAll)
		logger.Debug("msg")
		_, file, line, _ := runtime.Caller(1)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		want := fmt.Sprintf(`"level"="debug" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("level=info, callerDepth=1", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryAll, callerDepth: 1}), bb, VerbosityAll)
		logger.Info("msg")
		_, file, line, _ := runtime.Caller(1)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		want := fmt.Sprintf(`"level"="info" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("level=warning, callerDepth=1", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryAll, callerDepth: 1}), bb, VerbosityAll)
		logger.Warning("msg")
		_, file, line, _ := runtime.Caller(1)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		want := fmt.Sprintf(`"level"="warning" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg"`, filepath.Base(file), line)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
	t.Run("level=error, callerDepth=1", func(t *testing.T) {
		bb := new(bytes.Buffer)
		logger := newBasicLogger(newFormatter(fmtOptions{logCaller: categoryAll, callerDepth: 1}), bb, VerbosityAll)
		logger.Error(errors.New("err"), "msg")
		_, file, line, _ := runtime.Caller(1)
		have := string(bytes.TrimRight(bb.Bytes(), "\n"))
		want := fmt.Sprintf(`"level"="error" "logger"="root" "caller"={"file":%q,"line":%d} "msg"="msg" "error"="err"`, filepath.Base(file), line)
		if have != want {
			t.Errorf("\nwant %q\nhave %q", want, have)
		}
	})
}
