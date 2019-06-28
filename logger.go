package log4go

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// String 字符串输出
func (l level) String() string {
	if l < 0 || int(l) > len(levelStrings) {
		return "UNKNOWN"
	}
	return levelStrings[int(l)]
}

////////////////////////////////////////////////////////////////////////////////////

// SetExtend 设置扩展
func (record LogRecord) SetExtend(tp uint8, data []interface{}) {
	record.Extend = make([]interface{}, len(data)+1)
	record.Extend[0] = tp
	for index, info := range data {
		record.Extend[index+1] = info
	}
}

// GetExtend 获取扩展
func (record LogRecord) GetExtend() (tp uint8, data []interface{}) {
	tp = EXNone
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

////////////////////////////////////////////////////////////////////////////////////

// NewLogger 创建
func NewLogger() Logger {
	return make(Logger)
}

// Close 关闭
func (log Logger) Close() {
	for key, filt := range log {
		if filt != nil {
			filt.Close()
		}
		delete(log, key)
	}
}

// AddFilter 增加过滤器
func (log Logger) AddFilter(tag string, lvl level, writer LogWriter) Logger {
	log[tag] = &Filter{lvl, writer}
	return log
}

// checkSkip 检查
func (log Logger) checkSkip(lvl level) bool {
	for _, filt := range log {
		if filt != nil && lvl >= filt.Level {
			return false
		}
	}
	return true
}

// dispatchLog 分发日志
func (log Logger) dispatchLog(rec *LogRecord) {
	if rec != nil {
		for _, filt := range log {
			if filt != nil && rec.Level >= filt.Level {
				filt.LogWrite(rec)
			}
		}
	}
}

// getRunCaller 获取调用地址
func getRunCaller(skip int) string {
	pc, _, lineno, ok := runtime.Caller(skip + 1)
	if ok {
		return fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
	}
	return ""
}

// Send a formatted log message internally
func (log Logger) intLogf(skip int, lvl level, format string, args ...interface{}) {

	if log.checkSkip(lvl) == true {
		return
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  getRunCaller(skip + 1),
		Message: msg,
	})
}

// Send a closure log message internally
func (log Logger) intLogc(skip int, lvl level, closure func() string) {

	if log.checkSkip(lvl) == true {
		return
	}

	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  getRunCaller(skip + 1),
		Message: closure(),
	})
}

// Log Send a log message with manual level, source, and message.
func (log Logger) Log(lvl level, source, message string) {

	if log.checkSkip(lvl) == true {
		return
	}

	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  source,
		Message: message,
	})
}

// Logf format 日志输出
func (log Logger) Logf(lvl level, format string, args ...interface{}) {
	log.intLogf(1, lvl, format, args...)
}

// Logc closure 日志输出
func (log Logger) Logc(lvl level, closure func() string) {
	log.intLogc(1, lvl, closure)
}

// LogReport 上报
func (log Logger) LogReport(skip int, lvl level, url string, header interface{}, body string) {

	if log.checkSkip(lvl) == true {
		return
	}

	nurl := len(url) <= 0
	nhead := header == nil
	nbody := len(body) <= 0

	var extend []interface{}
	if nurl == false && nhead == false && nbody == false {
		extend = []interface{}{
			EXUrlHeadBody,
			url,
			header,
			body,
		}
	} else if nurl == false && nhead == false {
		extend = []interface{}{
			EXUrlHead,
			url,
			header,
		}
	} else if nurl == false && nbody == false {
		extend = []interface{}{
			EXUrlBody,
			url,
			body,
		}
	} else if nurl == false {
		extend = []interface{}{
			EXUrl,
			url,
		}
	} else {
		fmt.Fprintf(os.Stderr, "LogReport extend is nil, url: %s", url)
		return
	}

	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Extend:  extend,
		Source:  getRunCaller(skip + 1),
	})
}

// LogCmm 日志输出处理
func (log Logger) LogCmm(nerr bool, lvl level, arg0 interface{}, args ...interface{}) error {
	if nerr == false {
		switch first := arg0.(type) {
		case string:
			log.intLogf(2, lvl, first, args...)
		case func() string:
			log.intLogc(2, lvl, first)
		default:
			log.intLogf(2, lvl, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
		}
	} else {
		var msg string
		switch first := arg0.(type) {
		case string:
			msg = fmt.Sprintf(first, args...)
		case func() string:
			msg = first()
		default:
			msg = fmt.Sprintf(fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
		}
		log.intLogf(2, lvl, msg)
		return errors.New(msg)
	}
	return nil
}

// Finest logs
func (log Logger) Finest(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(false, FINEST, arg0, args...)
}

// Fine logs
func (log Logger) Fine(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(false, FINE, arg0, args...)
}

// Debug logs
func (log Logger) Debug(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(false, DEBUG, arg0, args...)
}

// Trace logs
func (log Logger) Trace(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(false, TRACE, arg0, args...)
}

// Info logs
func (log Logger) Info(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(false, INFO, arg0, args...)
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
func (log Logger) Report(arg0 interface{}, args ...interface{}) error {
	return log.LogCmm(false, REPORT, arg0, args...)
}

// ReportAPI Report Log by url
func (log Logger) ReportAPI(url string, header interface{}, body string) {
	log.LogReport(1, REPORT, url, header, body)
}
