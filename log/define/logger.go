package define

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////

// SetExtend 设置扩展
func (record LogRecord) SetExtend(tp uint8, data []interface{}) {
	record.Extend = make([]interface{}, 1, len(data)+1)
	record.Extend[0] = tp
	record.Extend = append(record.Extend, data...)
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
func (log Logger) AddFilter(tag string, writer LogWriter, lvl uint8) Logger {
	log[tag] = &Filter{writer, lvl}
	return log
}

// checkSkip 检查
func (log Logger) checkSkip(lvl uint8) bool {
	for _, filt := range log {
		if filt != nil && lvl >= filt.Level {
			return false
		}
	}
	return true
}

// checkReport 上报
func (log Logger) checkReport(rptp uint8) bool {
	if rptp <= 0 {
		return true
	}
	for _, filt := range log {
		if filt != nil && rptp == filt.GetReportType() {
			return true
		}
	}
	return false
}

// dispatchLog 分发日志
func (log Logger) dispatchLog(rec *LogRecord, rptp uint8) {
	if rec != nil {
		for _, filt := range log {
			if filt != nil && rec.Level >= filt.Level {
				if rptp > 0 && rptp != filt.GetReportType() {
					continue
				}
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
func (log Logger) intLogf(skip int, lvl uint8, format string, args ...interface{}) {
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
	}, 0)
}

// Log Send a log message with manual level, source, and message.
func (log Logger) Log(lvl uint8, source, message string) {
	if log.checkSkip(lvl) == true {
		return
	}
	log.dispatchLog(&LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  source,
		Message: message,
	}, 0)
}

// Logf format 日志输出
func (log Logger) Logf(lvl uint8, format string, args ...interface{}) {
	log.intLogf(1, lvl, format, args...)
}

// LogReport 上报
func (log Logger) LogReport(skip int, rptp, extp uint8, exdt ...interface{}) {
	if log.checkSkip(REPORT) == true || log.checkReport(rptp) == false {
		return
	}
	record := &LogRecord{
		Level:   REPORT,
		Created: time.Now(),
		Source:  getRunCaller(skip + 1),
	}
	if extp > 0 {
		record.SetExtend(extp, exdt)
	} else if len(exdt) > 0 {
		record.Message = exdt[0].(string)
	}
	log.dispatchLog(record, rptp)
}

func (log Logger) getArg(arg0 interface{}, larg int) string {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = first
	case func() string:
		msg = first()
	default:
		msg = fmt.Sprint(arg0) + strings.Repeat(" %v", larg)
	}
	return msg
}

// LogCmm 日志输出处理
func (log Logger) LogCmm(lvl uint8, arg0 interface{}, args ...interface{}) {
	log.Logf(lvl, log.getArg(arg0, len(args)), args...)
}

// Finest 最好log
func (log Logger) Finest(arg0 interface{}, args ...interface{}) {
	log.LogCmm(FINEST, arg0, args...)
}

// Fine 好log
func (log Logger) Fine(arg0 interface{}, args ...interface{}) {
	log.LogCmm(FINE, arg0, args...)
}

// Debug 调试log
func (log Logger) Debug(arg0 interface{}, args ...interface{}) {
	log.LogCmm(DEBUG, arg0, args...)
}

// Trace 追踪log
func (log Logger) Trace(arg0 interface{}, args ...interface{}) {
	log.LogCmm(TRACE, arg0, args...)
}

// Info 信息log
func (log Logger) Info(arg0 interface{}, args ...interface{}) {
	log.LogCmm(INFO, arg0, args...)
}

// Warn 警告log
func (log Logger) Warn(arg0 interface{}, args ...interface{}) {
	log.LogCmm(WARNING, arg0, args...)
}

// Error 错误log
func (log Logger) Error(arg0 interface{}, args ...interface{}) {
	log.LogCmm(ERROR, arg0, args...)
}

// Fatal 致命log
func (log Logger) Fatal(arg0 interface{}, args ...interface{}) {
	log.LogCmm(FATAL, arg0, args...)
}

// Report 上报log
func (log Logger) Report(rptp uint8, arg0 interface{}, args ...interface{}) {
	msg := log.getArg(arg0, len(args))
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	log.LogReport(1, rptp, 0, msg)
}

// Flume flume上报
func (log Logger) Flume(arg0 interface{}, args ...interface{}) {
	log.Report(FLUME, arg0, args...)
}

// FlumeAPI flume api上报
func (log Logger) FlumeAPI(url string, header interface{}, body interface{}) {
	log.LogReport(1, FLUME, EXUrlHeadBody, url, header, body)
}

// CatTransaction cat transaction支持
func (log Logger) CatTransaction(name string, status interface{}, data interface{}) {
	log.LogReport(1, CAT, EXCatTransaction, name, status, data)
}

// CatEvent cat event支持
func (log Logger) CatEvent(name string, status interface{}, data interface{}) {
	log.LogReport(1, CAT, EXCatEvent, name, status, data)
}

// CatError cat error支持
func (log Logger) CatError(name string, err interface{}) {
	log.LogReport(1, CAT, EXCatError, name, err)
}

// CatMetricCount cat metric count支持
func (log Logger) CatMetricCount(name string, count ...int) {
	if len(count) <= 0 {
		log.LogReport(1, CAT, EXCatMetricCount, name)
	} else {
		log.LogReport(1, CAT, EXCatMetricCount, name, count[0])
	}
}

// CatMetricDuration cat metric duration支持
func (log Logger) CatMetricDuration(name string, duration int64) {
	log.LogReport(1, CAT, EXCatMetricDuration, name, duration)
}
