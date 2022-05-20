// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"errors"
	"os"
	"sync"

	"github.com/go-logr/zapr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	zl := zerolog.New(os.Stderr).With().Caller().Timestamp().Logger()
	ll := zerologr.New(&zl)
	ll.Error(errors.New("new error"), "zerolog")
	ll.V(0).Info("zerolog")
	ll.V(1).Info("zerolog")
	ll.V(2).Info("zerolog")
	ll.V(3).Info("zerolog")
	ll.V(4).Info("zerolog")

	hooks := []func(zapcore.Entry) error{
		func(entry zapcore.Entry) error {
			switch entry.Level {
			case zapcore.ErrorLevel:
			case zapcore.WarnLevel:
			case zapcore.InfoLevel:
			case zapcore.DebugLevel:
			}
			return nil
		},
	}

	config := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	lvl := zap.NewAtomicLevelAt(zapcore.Level(-2))
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		os.Stderr,
		zapcore.DebugLevel,
	)
	logger = zap.New(zapcore.RegisterHooks(core, hooks...)).
		WithOptions(zap.IncreaseLevel(lvl), zap.AddCaller(), zap.AddCallerSkip(1)).
		Named(name)

	registry.levels[name] = lvl
	registry.loggers[name] = logger
	return &Logger{zapr.NewLogger(logger)}
}
