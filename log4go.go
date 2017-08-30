// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

// Package log4go provides level-based and highly configurable logging.
//
// Enhanced Logging
//
// This is inspired by the logging functionality in Java.  Essentially, you create a Logger
// object and create output filters for it.  You can send whatever you want to the Logger,
// and it will filter that based on your settings and send it to the outputs.  This way, you
// can put as much debug code in your program as you want, and when you're done you can filter
// out the mundane messages so only the important ones show up.
//
// Utility functions are provided to make life easier. Here is some example code to get started:
//
// log := log4go.NewLogger()
// log.AddFilter("stdout", log4go.DEBUG, log4go.NewConsoleLogWriter())
// log.AddFilter("log",    log4go.FINE,  log4go.NewFileLogWriter("example.log", true))
// log.Info("The time is now: %s", time.LocalTime().Format("15:04:05 MST 2006/01/02"))
//
// The first two lines can be combined with the utility NewDefaultLogger:
//
// log := log4go.NewDefaultLogger(log4go.DEBUG)
// log.AddFilter("log",    log4go.FINE,  log4go.NewFileLogWriter("example.log", true))
// log.Info("The time is now: %s", time.LocalTime().Format("15:04:05 MST 2006/01/02"))
//
// Usage notes:
// - The ConsoleLogWriter does not display the source of the message to standard
//   output, but the FileLogWriter does.
// - The utility functions (Info, Debug, Warn, etc) derive their source from the
//   calling function, and this incurs extra overhead.
//
// Changes from 2.0:
// - The external interface has remained mostly stable, but a lot of the
//   internals have been changed, so if you depended on any of this or created
//   your own LogWriter, then you will probably have to update your code.  In
//   particular, Logger is now a map and ConsoleLogWriter is now a channel
//   behind-the-scenes, and the LogWrite method no longer has return values.
//
// Future work: (please let me know if you think I should work on any of these particularly)
// - Log file rotation
// - Logging configuration files ala log4j
// - Have the ability to remove filters?
// - Have GetInfoChannel, GetDebugChannel, etc return a chan string that allows
//   for another method of logging
// - Add an XML filter type
package log4go

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// Version information
const (
	L4G_VERSION = "log4go-v3.0.1"
	L4G_MAJOR   = 3
	L4G_MINOR   = 0
	L4G_BUILD   = 1
)

/****** Constants ******/

// These are the integer logging levels used by the logger
type level int

const (
	FINEST level = iota
	FINE
	DEBUG
	TRACE
	INFO
	WARNING
	ERROR
	FATAL
	REPORT
)

// Logging level strings
var (
	levelStrings = [...]string{"fnst", "fine", "debug", "trace", "info", "warning", "error", "fatal", "report"}
)

func (l level) String() string {
	if l < 0 || int(l) > len(levelStrings) {
		return "UNKNOWN"
	}
	return levelStrings[int(l)]
}

/****** Variables ******/
var (
	// LogBufferLength specifies how many log messages a particular log4go
	// logger can buffer at a time before writing them.
	LogBufferLength = 32
)

const (
	EX_NONE uint8 = iota
	EX_URL
	EX_URL_HEAD
	EX_URL_BODY
	EX_URL_HEAD_BODY
)

/****** LogRecord ******/

// A LogRecord contains all of the pertinent information for each message
type LogRecord struct {
	Level   level     // The log level
	Created time.Time // The time at which the log message was created (nanoseconds)
	Source  string    // The message source
	Message string    // The log message
	Extend  []interface{}
}

func (record LogRecord) SetExtend(tp uint8, data []interface{}) {
	record.Extend = make([]interface{}, len(data)+1)
	record.Extend[0] = tp
	for index, info := range data {
		record.Extend[index+1] = info
	}
}

func (record LogRecord) GetExtend() (tp uint8, data []interface{}) {
	tp = EX_NONE
	data = nil
	elen := len(record.Extend)
	if elen > 0 {
		tp = record.Extend[0].(uint8)
		if elen > 1 {
			data = record.Extend[1:]
		}
	}
	return
}

/****** LogWriter ******/

// This is an interface for anything that should be able to write logs
type LogWriter interface {
	// This will be called to log a LogRecord message.
	LogWrite(rec *LogRecord)

	// This should clean up anything lingering about the LogWriter, as it is called before
	// the LogWriter is removed.  LogWrite should not be called after Close.
	Close()
}

/****** Logger ******/

// A Filter represents the log level below which no log records are written to
// the associated LogWriter.
type Filter struct {
	Level level
	LogWriter
}

// A Logger represents a collection of Filters through which log messages are
// written.
type Logger map[string]*Filter

// Create a new logger.
//
// DEPRECATED: Use make(Logger) instead.
func NewLogger() Logger {
	//os.Stderr.WriteString("warning: use of deprecated NewLogger\n")
	return make(Logger)
}

// Create a new logger with a "stdout" filter configured to send log messages at
// or above lvl to standard output.
//
// DEPRECATED: use NewDefaultLogger instead.
func NewConsoleLogger(lvl level) Logger {
	//os.Stderr.WriteString("warning: use of deprecated NewConsoleLogger\n")
	return Logger{
		"stdout": &Filter{lvl, NewConsoleLogWriter()},
	}
}

// Create a new logger with a "stdout" filter configured to send log messages at
// or above lvl to standard output.
func NewDefaultLogger(lvl level) Logger {
	return Logger{
		"stdout": &Filter{lvl, NewConsoleLogWriter()},
	}
}

// Closes all log writers in preparation for exiting the program or a
// reconfiguration of logging.  Calling this is not really imperative, unless
// you want to guarantee that all log messages are written.  Close removes
// all filters (and thus all LogWriters) from the logger.
func (log Logger) Close() {
	// Close all open loggers
	for name, filt := range log {
		filt.Close()
		delete(log, name)
	}
}

// Add a new LogWriter to the Logger which will only log messages at lvl or
// higher.  This function should not be called from multiple goroutines.
// Returns the logger for chaining.
func (log Logger) AddFilter(name string, lvl level, writer LogWriter) Logger {
	log[name] = &Filter{lvl, writer}
	return log
}

// check skip
func (log Logger) checkSkip(lvl level) bool {
	// Determine if any logging will be done
	for _, filt := range log {
		if lvl >= filt.Level {
			return false
		}
	}
	return true
}

// dispatch log
func (log Logger) dispatchLog(rec *LogRecord) {
	if rec != nil {
		eqrep := (rec.Level == REPORT)
		for _, filt := range log {
			if (eqrep == true && rec.Level == filt.Level) ||
				(eqrep == false && rec.Level >= filt.Level && filt.Level != REPORT) {
				filt.LogWrite(rec)
			}
		}
	}
}

/******* Logging *******/
func getRunCaller(skip int) string {
	pc, _, lineno, ok := runtime.Caller(skip + 1)
	if ok {
		return fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
	}
	return ""
}

// Send a formatted log message internally
func (log Logger) intLogf(skip int, lvl level, format string, args ...interface{}) {
	// check skip
	if log.checkSkip(lvl) == true {
		return
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	//dispatch log
	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  getRunCaller(skip + 1),
		Message: msg,
	})
}

// Send a closure log message internally
func (log Logger) intLogc(skip int, lvl level, closure func() string) {
	// check skip
	if log.checkSkip(lvl) == true {
		return
	}

	// dispatch log
	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  getRunCaller(skip + 1),
		Message: closure(),
	})
}

// Send a log message with manual level, source, and message.
func (log Logger) Log(lvl level, source, message string) {
	// check skip
	if log.checkSkip(lvl) == true {
		return
	}

	// dispatch log
	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  source,
		Message: message,
	})
}

// Logf logs a formatted log message at the given log level, using the caller as
// its source.
func (log Logger) Logf(lvl level, format string, args ...interface{}) {
	log.intLogf(1, lvl, format, args...)
}

// Logc logs a string returned by the closure at the given log level, using the caller as
// its source.  If no log message would be written, the closure is never called.
func (log Logger) Logc(lvl level, closure func() string) {
	log.intLogc(1, lvl, closure)
}

func (log Logger) LogReport(skip int, lvl level, url string, header interface{}, body string) {
	// check skip
	if log.checkSkip(lvl) == true {
		return
	}

	nurl := len(url) <= 0
	nhead := header == nil
	nbody := len(body) <= 0

	var extend []interface{}
	if nurl == false && nhead == false && nbody == false {
		extend = []interface{}{
			EX_URL_HEAD_BODY,
			url,
			header,
			body,
		}
	} else if nurl == false && nhead == false {
		extend = []interface{}{
			EX_URL_HEAD,
			url,
			header,
		}
	} else if nurl == false && nbody == false {
		extend = []interface{}{
			EX_URL_BODY,
			url,
			body,
		}
	} else if nurl == false {
		extend = []interface{}{
			EX_URL,
			url,
		}
	} else {
		fmt.Fprintf(os.Stderr, "LogReport extend is nil, url: %s", url)
		return
	}

	// dispatch log
	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Extend:  extend,
		Source:  getRunCaller(skip + 1),
	})
}

// comm func
// nerr means whether need return error
// level means log level
// The behavior of Debug depends on the first argument:
// - arg0 is a string
//   When given a string as the first argument, this behaves like Logf but with
//   the DEBUG log level: the first argument is interpreted as a format for the
//   latter arguments.
// - arg0 is a func()string
//   When given a closure of type func()string, this logs the string returned by
//   the closure iff it will be logged.  The closure runs at most one time.
// - arg0 is interface{}
//   When given anything else, the log message will be each of the arguments
//   formatted with %v and separated by spaces (ala Sprint).
func (log Logger) LogCmm(nerr bool, lvl level, arg0 interface{}, args ...interface{}) error {
	if nerr == false {
		switch first := arg0.(type) {
		case string:
			// Use the string as a format string
			log.intLogf(2, lvl, first, args...)
		case func() string:
			// Log the closure (no other arguments used)
			log.intLogc(2, lvl, first)
		default:
			// Build a format string so that it will be similar to Sprint
			log.intLogf(2, lvl, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
		}
	} else {
		var msg string
		switch first := arg0.(type) {
		case string:
			// Use the string as a format string
			msg = fmt.Sprintf(first, args...)
		case func() string:
			// Log the closure (no other arguments used)
			msg = first()
		default:
			// Build a format string so that it will be similar to Sprint
			msg = fmt.Sprintf(fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
		}
		log.intLogf(2, lvl, msg)
		return errors.New(msg)
	}
	return nil
}

// Finest logs
func (log Logger) Finest(arg0 interface{}, args ...interface{}) {
	log.LogCmm(false, FINEST, arg0, args...)
}

// Fine logs
func (log Logger) Fine(arg0 interface{}, args ...interface{}) {
	log.LogCmm(false, FINE, arg0, args...)
}

// Debug logs
func (log Logger) Debug(arg0 interface{}, args ...interface{}) {
	log.LogCmm(false, DEBUG, arg0, args...)
}

// Trace logs
func (log Logger) Trace(arg0 interface{}, args ...interface{}) {
	log.LogCmm(false, TRACE, arg0, args...)
}

// Info logs
func (log Logger) Info(arg0 interface{}, args ...interface{}) {
	log.LogCmm(false, INFO, arg0, args...)
}

// Warn logs
func (log Logger) Warn(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(true, WARNING, arg0, args...)
}

// Error logs
func (log Logger) Error(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(true, ERROR, arg0, args...)
}

// Fatal logs
func (log Logger) Fatal(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(true, FATAL, arg0, args...)
}

// Report logs
func (log Logger) Report(arg0 interface{}, args ...interface{}) {
	log.LogCmm(false, REPORT, arg0, args...)
}

// Report Log by url
func (log Logger) ReportAPI(url string, header interface{}, body string) {
	log.LogReport(1, REPORT, url, header, body)
}
