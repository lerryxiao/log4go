package log4go

import (
	"time"
)

// Version information
const (
	L4GVersion = "log4go-v3.0.1"
	L4GMajor   = 3
	L4GMinor   = 0
	L4GBuild   = 1

	LogBufferLength = 32 // logger can buffer at a time before writing them.
)

type level uint8

// level 定义
const (
	_ level = iota
	FINEST
	FINE
	DEBUG
	TRACE
	INFO
	WARNING
	ERROR
	FATAL
	REPORT
)

// 扩展定义
const (
	EXNone uint8 = iota
	EXUrl
	EXUrlHead
	EXUrlBody
	EXUrlHeadBody
)

// Logging level strings
var (
	levelStrings = []string{"fnst", "fine", "debug", "trace", "info", "warning", "error", "fatal", "report"}
)

// A LogRecord contains all of the pertinent information for each message
type LogRecord struct {
	Level   level     // The log level
	Created time.Time // The time at which the log message was created (nanoseconds)
	Source  string    // The message source
	Message string    // The log message
	Extend  []interface{}
}

// LogWriter 日志输出器
type LogWriter interface {
	LogWrite(rec *LogRecord)
	Close()
}

// Filter 日志过滤器
type Filter struct {
	Level level
	LogWriter
}

// Logger 日志过滤器组合
type Logger map[string]*Filter
