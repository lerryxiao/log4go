package log4go

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// 常量定义
const (
	FormatDefault = "[%D %T] [%L] (%S) %M"
	FormatShort   = "[%t %d] [%L] %M"
	FormatAbbrev  = "[%L] %M"
)

type formatCacheType struct {
	LastUpdateSeconds     int64
	shortTime, shortDate  string
	longTime, longDate    string
	runnableID, processID string
}

var (
	formatCache = &formatCacheType{}
)

// FormatLogRecord Known format codes:
// %T - Time (15:04:05 MST)
// %t - Time (15:04)
// %D - Date (2006-01-02)
// %d - Date (01/02/06)
// %L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
// %S - Source
// %M - Message
// Ignores unknown formats
// Recommended: "[%D %T] [%L] (%S) %M"
// add by format by lerry 2015-06-30
// %P - PROCESS ID
// %R - thread ID
func FormatLogRecord(format string, rec *LogRecord) string {
	if rec == nil {
		return "<nil>"
	}
	if len(format) == 0 {
		return ""
	}

	out := bytes.NewBuffer(make([]byte, 0, 64))
	secs := rec.Created.UnixNano() / 1e9

	cache := *formatCache
	if cache.LastUpdateSeconds != secs {
		month, day, year := rec.Created.Month(), rec.Created.Day(), rec.Created.Year()
		hour, minute, second := rec.Created.Hour(), rec.Created.Minute(), rec.Created.Second()
		//zone, _ := rec.Created.Zone()
		millis := rec.Created.UnixNano()/1000000 - (secs * 1000)
		updated := &formatCacheType{
			LastUpdateSeconds: secs,
			shortTime:         fmt.Sprintf("%02d:%02d", hour, minute),
			shortDate:         fmt.Sprintf("%02d/%02d/%02d", month, day, year%100),
			longTime:          fmt.Sprintf("%02d:%02d:%02d:%03d", hour, minute, second, millis),
			longDate:          fmt.Sprintf("%04d-%02d-%02d", year, month, day),
			runnableID:        fmt.Sprintf("%d", 0),
			processID:         fmt.Sprintf("%d", os.Getpid()),
		}
		cache = *updated
		formatCache = updated
	}

	// Split the string into pieces by % signs
	pieces := bytes.Split([]byte(format), []byte{'%'})

	// Iterate over the pieces, replacing known formats
	for i, piece := range pieces {
		if i > 0 && len(piece) > 0 {
			switch piece[0] {
			case 'T':
				out.WriteString(cache.longTime)
			case 't':
				out.WriteString(cache.shortTime)
			case 'D':
				out.WriteString(cache.longDate)
			case 'd':
				out.WriteString(cache.shortDate)
			case 'L':
				out.WriteString(levelStrings[rec.Level])
			case 'S':
				out.WriteString(rec.Source)
			case 'M':
				out.WriteString(rec.Message)
			//add by lerry suport processID and threadID ,but threadID is 0
			case 'P':
				out.WriteString(cache.processID)
			case 'R':
				out.WriteString(cache.runnableID)
			}
			if len(piece) > 1 {
				out.Write(piece[1:])
			}
		} else if len(piece) > 0 {
			out.Write(piece)
		}
	}
	out.WriteByte('\n')

	return out.String()
}

// FormatLogWriter This is the standard writer that prints to standard output.
type FormatLogWriter struct {
	rec  chan *LogRecord
	stop chan bool
}

// NewFormatLogWriter This creates a new FormatLogWriter
func NewFormatLogWriter(out io.Writer, format string) *FormatLogWriter {
	w := &FormatLogWriter{
		rec:  make(chan *LogRecord, LogBufferLength),
		stop: make(chan bool),
	}

	go func() {
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
					if ok == true {
						fmt.Fprint(out, FormatLogRecord(format, rec))
					}
				}
			}
		}
	EXIT:
	}()

	return w
}

// LogWrite This is the FormatLogWriter's output method.  This will block if the output buffer is full.
func (w *FormatLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

// Close stops the logger from sending messages to standard output.  Attempts to
// send log messages to this logger after a Close have undefined behavior.
func (w *FormatLogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}
