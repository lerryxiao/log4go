package define

import (
	"time"
)

// Version information
const (
	LogBufferLength = 32 // logger can buffer at a time before writing them.
)

// level 定义
const (
	_       uint8 = iota
	FINEST        // 最好
	FINE          // 好
	DEBUG         // 调试
	TRACE         // 追踪
	INFO          // 信息
	WARNING       // 警告
	ERROR         // 错误
	FATAL         // 致命错误
	REPORT        // 上报
)

// 上报定义
const (
	_     uint8 = iota
	FLUME       // flume上报
	CAT         // cat追踪
	MAX
)

// 扩展定义
const (
	EXNone              uint8 = iota
	EXUrlHeadBody             // url header body 上报
	EXCatTransaction          // cat事务
	EXCatEvent                // cat事件
	EXCatError                // cat错误
	EXCatMetricCount          // cat调用次数
	EXCatMetricDuration       // cat调用时间
)

// Logging level strings
var (
	LevelStrings = []string{"", "fnst", "fine", "debug", "trace", "info", "warning", "error", "fatal", "report"}
)

// LogRecord contains all of the pertinent information for each message
type LogRecord struct {
	Level   uint8     // The log level
	Created time.Time // The time at which the log message was created (nanoseconds)
	Source  string    // The message source
	Message string    // The log message
	Extend  []interface{}
}

// LogWriter 日志输出器
type LogWriter interface {
	LogWrite(rec *LogRecord)
	SetReportType(uint8)
	GetReportType() uint8
	Close()
}

// WriterCreater 创建函数
type WriterCreater = func(string, []XMLProperty) (LogWriter, bool)

// Filter 日志过滤器
type Filter struct {
	LogWriter
	Level uint8
}

// Logger 日志过滤器组合
type Logger map[string]*Filter
