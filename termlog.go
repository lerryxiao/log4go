package log4go

import (
	"fmt"
	"io"
	"os"
)

var (
	stdout = os.Stdout
)

// ConsoleLogWriter This is the standard writer that prints to standard output.
type ConsoleLogWriter struct {
	rec  chan *LogRecord
	stop chan bool
}

// NewConsoleLogWriter This creates a new ConsoleLogWriter
func NewConsoleLogWriter() *ConsoleLogWriter {
	w := &ConsoleLogWriter{
		rec:  make(chan *LogRecord, LogBufferLength),
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
				fmt.Fprint(out, "[", timestr, "] [", levelStrings[rec.Level], "] ", rec.Message, "\n")
			}
		}
	}
EXIT:
}

// LogWrite This is the ConsoleLogWriter's output method.  This will block if the output buffer is full.
func (w *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

// Close stops the logger from sending messages to standard output.  Attempts to
// send log messages to this logger after a Close have undefined behavior.
func (w *ConsoleLogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}
