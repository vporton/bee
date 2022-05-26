// Copyright 2022 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Note: the following code is derived (borrows) from: github.com/go-logr/logr

package log

import (
	"bytes"
	"encoding"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Marshaler is an optional interface that logged values may choose to
// implement. Loggers with structured output, such as JSON, should
// log the object return by the MarshalLog method instead of the
// original value.
type Marshaler interface {
	// MarshalLog can be used to:
	//   - ensure that structs are not logged as strings when the original
	//     value has a String method: return a different type without a
	//     String method
	//   - select which fields of a complex type should get logged:
	//     return a simpler struct with fewer fields
	//   - log unexported fields: return a different struct
	//     with exported fields
	//
	// It may return any value of any type.
	MarshalLog() interface{}
}

// fmtOptions carries parameters which influence the way logs are generated.
type fmtOptions struct {
	// logCaller tells formatter to add a "caller" key to some or all log lines.
	// This has some overhead, so some users might not want it.
	logCaller messageCategory

	// logCallerFunc tells formatter to also log the calling function name.
	// This has no effect if caller logging is not enabled (see fmtOptions.logCaller).
	logCallerFunc bool

	// logTimestamp tells formatter to add a "timestamp" key to log lines.
	// This has some overhead, so some users might not want it.
	logTimestamp bool

	// timestampFormat tells formatter how to render timestamps when
	// fmtOptions.logTimestamp is enabled. If not specified, a default
	// format will be used. For more details, see docs for Go's time.Layout.
	timestampFormat string

	// maxLogDepth tells formatter how many levels of nested fields
	// (e.g. a struct that contains a struct, etc.) it may log. Every time
	// it finds a struct, slice, array, or map the depth is increased by one.
	// When the maximum is reached, the value will be converted to a string
	// indicating that the max depth has been exceeded. If this field is not
	// specified, a default value will be used.
	maxLogDepth int

	// outputFormat tells formatter which outputFormat to use. If this field
	// is not specified, a default value outputKeyValue will be used.
	outputFormat outputFormat

	// callerDepth tells formatter the number of stack-frames to skip
	// when attributing the log line to a file and line.
	callerDepth int
}

// messageCategory indicates which category or categories of messages to consider.
type messageCategory int

const (
	categoryNone messageCategory = iota
	categoryAll
	categoryError
	categoryWarning
	categoryInfo
	categoryDebug
)

// noValue is a placeholder for missing values.
const noValue = "<no-value>"

// Defaults for fmtOptions.
const defaultTimestampFormat = "2006-01-02 15:04:05.000000"
const defaultMaxLogDepth = 16

// outputFormat indicates which outputFormat to use.
type outputFormat int

const (
	// outputKeyValue emits a JSON-like key=value format, but not strict JSON.
	outputKeyValue outputFormat = iota
	// outputJSON emits strict JSON.
	outputJSON
)

// PseudoStruct is a list of key-value pairs that gets logged as a struct.
type PseudoStruct []interface{}

// Caller represents the original call site for a log line, after considering
// logr.Logger.WithCallDepth and logr.Logger.WithCallStackHelper.  The File and
// Line fields will always be provided, while the Func field is optional.
// Users can set the render hook fields in fmtOptions to examine logged key-value
// pairs, one of which will be {"caller", Caller} if the fmtOptions.logCaller
// field is enabled for the given messageCategory.
type Caller struct {
	// File is the basename of the file for this call site.
	File string `json:"file"`
	// Line is the line number in the file for this call site.
	Line int `json:"line"`
	// Func is the function name for this call site, or empty if
	// fmtOptions.logCallerFunc is not enabled.
	Func string `json:"function,omitempty"`
}

// constructs a formatter which influenced by given options.
func newFormatter(opts fmtOptions) Formatter {
	if opts.timestampFormat == "" {
		opts.timestampFormat = defaultTimestampFormat
	}
	if opts.maxLogDepth == 0 {
		opts.maxLogDepth = defaultMaxLogDepth
	}
	return Formatter{
		prefix: "root",
		values: nil,
		opts:   opts,
	}
}

// Formatter is an opaque struct which can be embedded in a LogSink
// implementation. It should be constructed with NewFormatter. Some of
// its methods directly implement logr.LogSink.
type Formatter struct {
	prefix    string
	values    []interface{} // TODO: do deep copy on log clone!?
	valuesStr string
	opts      fmtOptions
}

// render produces a log line.
func (f Formatter) render(builtins, args []interface{}) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	if f.opts.outputFormat == outputJSON {
		buf.WriteByte('{')
	}

	f.flatten(buf, builtins, false, false)
	continuing := len(builtins) > 0
	if len(f.valuesStr) > 0 {
		if continuing {
			if f.opts.outputFormat == outputJSON {
				buf.WriteByte(',')
			} else {
				buf.WriteByte(' ')
			}
		}
		continuing = true
		buf.WriteString(f.valuesStr)
	}

	f.flatten(buf, args, continuing, true)
	if f.opts.outputFormat == outputJSON {
		buf.WriteByte('}')
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

// flatten renders a list of key-value pairs into a buffer.  If continuing is
// true, it assumes that the buffer has previous values and will emit a
// separator (which depends on the output format) before the first pair it
// writes.  If escapeKeys is true, the keys are assumed to have
// non-JSON-compatible characters in them and must be evaluated for escapes.
//
// This function returns a potentially modified version of kvList, which
// ensures that there is a value for every key (adding a value if needed) and
// that each key is a string (substituting a key if needed).
func (f Formatter) flatten(buf *bytes.Buffer, kvList []interface{}, continuing bool, escapeKeys bool) []interface{} {
	// This logic overlaps with sanitize() but saves one type-cast per key,
	// which can be measurable.
	if len(kvList)%2 != 0 {
		kvList = append(kvList, noValue)
	}
	for i := 0; i < len(kvList); i += 2 {
		k, ok := kvList[i].(string)
		if !ok {
			k = f.nonStringKey(kvList[i])
			kvList[i] = k
		}
		v := kvList[i+1]

		if i > 0 || continuing {
			if f.opts.outputFormat == outputJSON {
				buf.WriteByte(',')
			} else {
				// In theory the format could be something we don't understand.  In
				// practice, we control it, so it won't be.
				buf.WriteByte(' ')
			}
		}

		if escapeKeys {
			buf.WriteString(prettyString(k))
		} else {
			// this is faster
			buf.WriteByte('"')
			buf.WriteString(k)
			buf.WriteByte('"')
		}
		if f.opts.outputFormat == outputJSON {
			buf.WriteByte(':')
		} else {
			buf.WriteByte('=')
		}
		buf.WriteString(f.pretty(v))
	}
	return kvList
}

func (f Formatter) pretty(value interface{}) string {
	return f.prettyWithFlags(value, 0, 0)
}

// TODO: This is not fast. Most of the overhead goes here.
func (f Formatter) prettyWithFlags(value interface{}, flags uint32, depth int) string {
	const flagRawStruct = 0x1 // Do not print braces on structs.

	if depth > f.opts.maxLogDepth {
		return `"<max-log-depth-exceeded>"`
	}

	// Handle types that take full control of logging.
	if v, ok := value.(Marshaler); ok {
		// Replace the value with what the type wants to get logged.
		// That then gets handled below via reflection.
		value = invokeMarshaler(v)
	}

	// Handle types that want to format themselves.
	switch v := value.(type) {
	case fmt.Stringer:
		value = invokeStringer(v)
	case error:
		value = invokeError(v)
	}

	// Handling the most common types without reflect is a small perf win.
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case string:
		return prettyString(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uintptr:
		return strconv.FormatUint(uint64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case complex64:
		return `"` + strconv.FormatComplex(complex128(v), 'f', -1, 64) + `"`
	case complex128:
		return `"` + strconv.FormatComplex(v, 'f', -1, 128) + `"`
	case PseudoStruct:
		buf := bytes.NewBuffer(make([]byte, 0, 1024))
		v = f.sanitize(v)
		if flags&flagRawStruct == 0 {
			buf.WriteByte('{')
		}
		for i := 0; i < len(v); i += 2 {
			if i > 0 {
				buf.WriteByte(',')
			}
			k, _ := v[i].(string) // sanitize() above means no need to check success
			// arbitrary keys might need escaping
			buf.WriteString(prettyString(k))
			buf.WriteByte(':')
			buf.WriteString(f.prettyWithFlags(v[i+1], 0, depth+1))
		}
		if flags&flagRawStruct == 0 {
			buf.WriteByte('}')
		}
		return buf.String()
	}

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	t := reflect.TypeOf(value)
	if t == nil {
		return "null"
	}
	v := reflect.ValueOf(value)
	switch t.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return prettyString(v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(int64(v.Int()), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(uint64(v.Uint()), 10)
	case reflect.Float32:
		return strconv.FormatFloat(float64(v.Float()), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Complex64:
		return `"` + strconv.FormatComplex(complex128(v.Complex()), 'f', -1, 64) + `"`
	case reflect.Complex128:
		return `"` + strconv.FormatComplex(v.Complex(), 'f', -1, 128) + `"`
	case reflect.Struct:
		if flags&flagRawStruct == 0 {
			buf.WriteByte('{')
		}
		for i := 0; i < t.NumField(); i++ {
			fld := t.Field(i)
			if fld.PkgPath != "" {
				// reflect says this field is only defined for non-exported fields.
				continue
			}
			if !v.Field(i).CanInterface() {
				// reflect isn't clear exactly what this means, but we can't use it.
				continue
			}
			name := ""
			omitempty := false
			if tag, found := fld.Tag.Lookup("json"); found {
				if tag == "-" {
					continue
				}
				if comma := strings.Index(tag, ","); comma != -1 {
					if n := tag[:comma]; n != "" {
						name = n
					}
					rest := tag[comma:]
					if strings.Contains(rest, ",omitempty,") || strings.HasSuffix(rest, ",omitempty") {
						omitempty = true
					}
				} else {
					name = tag
				}
			}
			if omitempty && isEmpty(v.Field(i)) {
				continue
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			if fld.Anonymous && fld.Type.Kind() == reflect.Struct && name == "" {
				buf.WriteString(f.prettyWithFlags(v.Field(i).Interface(), flags|flagRawStruct, depth+1))
				continue
			}
			if name == "" {
				name = fld.Name
			}
			// field names can't contain characters which need escaping
			buf.WriteByte('"')
			buf.WriteString(name)
			buf.WriteByte('"')
			buf.WriteByte(':')
			buf.WriteString(f.prettyWithFlags(v.Field(i).Interface(), 0, depth+1))
		}
		if flags&flagRawStruct == 0 {
			buf.WriteByte('}')
		}
		return buf.String()
	case reflect.Slice, reflect.Array:
		buf.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			e := v.Index(i)
			buf.WriteString(f.prettyWithFlags(e.Interface(), 0, depth+1))
		}
		buf.WriteByte(']')
		return buf.String()
	case reflect.Map:
		buf.WriteByte('{')
		// This does not sort the map keys, for best perf.
		it := v.MapRange()
		i := 0
		for it.Next() {
			if i > 0 {
				buf.WriteByte(',')
			}
			// If a map key supports TextMarshaler, use it.
			keystr := ""
			if m, ok := it.Key().Interface().(encoding.TextMarshaler); ok {
				txt, err := m.MarshalText()
				if err != nil {
					keystr = fmt.Sprintf("<error-MarshalText: %s>", err.Error())
				} else {
					keystr = string(txt)
				}
				keystr = prettyString(keystr)
			} else {
				// prettyWithFlags will produce already-escaped values
				keystr = f.prettyWithFlags(it.Key().Interface(), 0, depth+1)
				if t.Key().Kind() != reflect.String {
					// JSON only does string keys.  Unlike Go's standard JSON, we'll
					// convert just about anything to a string.
					keystr = prettyString(keystr)
				}
			}
			buf.WriteString(keystr)
			buf.WriteByte(':')
			buf.WriteString(f.prettyWithFlags(it.Value().Interface(), 0, depth+1))
			i++
		}
		buf.WriteByte('}')
		return buf.String()
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return "null"
		}
		return f.prettyWithFlags(v.Elem().Interface(), 0, depth)
	}
	return fmt.Sprintf(`"<unhandled-%s>"`, t.Kind().String())
}

func prettyString(s string) string {
	// Avoid escaping (which does allocations) if we can.
	if needsEscape(s) {
		return strconv.Quote(s)
	}
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteByte('"')
	b.WriteString(s)
	b.WriteByte('"')
	return b.String()
}

// needsEscape determines whether the input string needs to be escaped or not,
// without doing any allocations.
func needsEscape(s string) bool {
	for _, r := range s {
		if !strconv.IsPrint(r) || r == '\\' || r == '"' {
			return true
		}
	}
	return false
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func invokeMarshaler(m Marshaler) (ret interface{}) {
	defer func() {
		if r := recover(); r != nil {
			ret = fmt.Sprintf("<panic: %s>", r)
		}
	}()
	return m.MarshalLog()
}

func invokeStringer(s fmt.Stringer) (ret string) {
	defer func() {
		if r := recover(); r != nil {
			ret = fmt.Sprintf("<panic: %s>", r)
		}
	}()
	return s.String()
}

func invokeError(e error) (ret string) {
	defer func() {
		if r := recover(); r != nil {
			ret = fmt.Sprintf("<panic: %s>", r)
		}
	}()
	return e.Error()
}

func (f Formatter) caller() Caller {
	pc, file, line, ok := runtime.Caller(f.opts.callerDepth + 2)
	if !ok {
		return Caller{"<unknown>", 0, ""}
	}
	caller := Caller{File: filepath.Base(file), Line: line}
	if f.opts.logCallerFunc {
		if fp := runtime.FuncForPC(pc); fp != nil {
			caller.Func = fp.Name()
		}
	}
	return caller
}

func (f Formatter) nonStringKey(v interface{}) string {
	return fmt.Sprintf("<non-string-key: %s>", f.snippet(v))
}

// snippet produces a short snippet string of an arbitrary value.
func (f Formatter) snippet(v interface{}) string {
	const snipLen = 16

	snip := f.pretty(v)
	if len(snip) > snipLen {
		snip = snip[:snipLen]
	}
	return snip
}

// sanitize ensures that a list of key-value pairs has a value for every key
// (adding a value if needed) and that each key is a string (substituting a key
// if needed).
func (f Formatter) sanitize(kvList []interface{}) []interface{} {
	if len(kvList)%2 != 0 {
		kvList = append(kvList, noValue)
	}
	for i := 0; i < len(kvList); i += 2 {
		_, ok := kvList[i].(string)
		if !ok {
			kvList[i] = f.nonStringKey(kvList[i])
		}
	}
	return kvList
}

func (f Formatter) base(level string) []interface{} {
	args := make([]interface{}, 0, 64) // using a constant here impacts perf
	if f.opts.logTimestamp {
		args = append(args, "time", time.Now().Format(f.opts.timestampFormat))
	}
	return append(args, "level", level, "logger", f.prefix)
}

// AddName appends the specified name.  formatter uses '/' characters to separate
// name elements.  Callers should not pass '/' in the provided name string, but
// this library does not actually enforce that.
func (f *Formatter) AddName(name string) {
	if len(f.prefix) > 0 {
		f.prefix += "/"
	}
	f.prefix += name
}

// AddValues adds key-value pairs to the set of saved values to be logged with
// each log line.
func (f *Formatter) AddValues(kvList []interface{}) {
	// Three slice args forces a copy.
	n := len(f.values)
	f.values = append(f.values[:n:n], kvList...)

	// Pre-render values, so we don't have to do it on each call.
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	f.flatten(buf, f.values, false, true) // escape user-provided keys
	f.valuesStr = buf.String()
}
