package log4go

import (
	"github.com/lerryxiao/log4go/log"
	"github.com/lerryxiao/log4go/log/define"
)

// 常量定义
const (
	FINEST  = define.FINEST
	FINE    = define.FINE
	DEBUG   = define.DEBUG
	TRACE   = define.TRACE
	INFO    = define.INFO
	WARNING = define.WARNING
	ERROR   = define.ERROR
	FATAL   = define.FATAL
	REPORT  = define.REPORT

	FLUME = define.FLUME
	CAT   = define.CAT

	EXNone              = define.EXNone
	EXUrlHeadBody       = define.EXUrlHeadBody
	EXCatTransaction    = define.EXCatTransaction
	EXCatEvent          = define.EXCatEvent
	EXCatError          = define.EXCatError
	EXCatMetricCount    = define.EXCatMetricCount
	EXCatMetricDuration = define.EXCatMetricDuration
)

// 函数定义
var (
	NewFileLogWriter    = log.NewFileLogWriter
	NewHTTPLogWriter    = log.NewHTTPLogWriter
	NewFormatLogWriter  = log.NewFormatLogWriter
	NewSocketLogWriter  = log.NewSocketLogWriter
	NewConsoleLogWriter = log.NewConsoleLogWriter
)

// Logger 日志过滤器组合
type Logger = define.Logger

// LogRecord contains all of the pertinent information for each message
type LogRecord = define.LogRecord
