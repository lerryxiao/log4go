package log

import (
	"fmt"
	"io"
	"os"
	"github.com/lerryxiao/log4go/log/define"
)

var (
	stdout = os.Stdout
)

// ConsoleLogWriter 控制台日志输出
type ConsoleLogWriter struct {
	rec    chan *LogRecord
	stop   chan bool
	rptype uint8
}

// NewConsoleLogWriter 创建控制台日志输出
func NewConsoleLogWriter() *ConsoleLogWriter {
	w := &ConsoleLogWriter{
		rec:  make(chan *LogRecord, define.LogBufferLength),
		stop: make(chan bool),
	}
	go w.run(stdout)
	return w
}

func (w ConsoleLogWriter) run(out io.Writer) {
	var timestr string
	var timestrAt int64

	defer func() {
		w.stop <- true
	}()

	for {
		select {
		case <-w.stop:
			{
				goto EXIT
			}
		case rec, ok := <-w.rec:
			{
				if ok == false {
					goto EXIT
				}
				if rec == nil {
					continue
				}
				if at := rec.Created.UnixNano() / 1e9; at != timestrAt {
					timestr, timestrAt = rec.Created.Format("01/02/06 15:04:05"), at
				}
				fmt.Fprint(out, "[", timestr, "] [", define.LevelStrings[rec.Level], "] ", rec.Message, "\n")
			}
		}
	}
EXIT:
}

// LogWrite 日志输出
func (w *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

// Close 关闭
func (w *ConsoleLogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}

// SetReportType 设置上报类型
func (w *ConsoleLogWriter) SetReportType(tp uint8) {
	w.rptype = tp
}

// GetReportType 获取上报类型
func (w *ConsoleLogWriter) GetReportType() uint8 {
	return w.rptype
}

// XMLToConsoleLogWriter xml创建控制台日志输出
func XMLToConsoleLogWriter(filename string, props []define.XMLProperty) (LogWriter, bool) {
	return NewConsoleLogWriter(), true
}
