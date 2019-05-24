package log4go

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"
)

// FileLogWriter This log writer sends output to a file dir a
type FileLogWriter struct {
	rec  chan *LogRecord
	rot  chan bool
	stop chan bool

	// The opened file
	dir            string
	filename       string
	filenameFormat string
	file           *os.File

	// The logging format
	format string

	// File header/trailer
	header, trailer string

	// Rotate at linecount
	maxlines         int
	maxlinesCurlines int

	// Rotate at size
	maxsize        int
	maxsizeCursize int

	// Rotate daily
	daily         bool
	dailyOpendate int

	// Keep old logfiles (.001, .002, etc)
	rotate bool
}

// LogWrite This is the FileLogWriter's output method
func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

// Close 关闭
func (w *FileLogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}

// NewFileLogWriter creates a new LogWriter which writes to the given file and has rotation enabled if rotate is true.
//
// If rotate is true, any time a new log file is opened, the old one is renamed with a .### extension to preserve it.
// The various Set* methods can be used to configure log rotation based on lines, size, and daily.
//
// The standard log-line format is: [%D %T] [%L] (%S) %M
func NewFileLogWriter(dir, fname string, rotate bool) *FileLogWriter {
	w := &FileLogWriter{
		rec:            make(chan *LogRecord, LogBufferLength),
		rot:            make(chan bool),
		stop:           make(chan bool),
		dir:            dir,
		filename:       fname,
		filenameFormat: fname,
		format:         "[%D %T] [%L] (%S) %M",
		rotate:         rotate,
	}
	//check dir is exist,
	if len(strings.TrimSpace(dir)) > 0 {
		_, err := os.Stat(dir)
		if err != nil {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FilleLogWriter(%q): %s ", dir, err)
				return nil
			}
		}

	}

	// open the file for the first time
	if err := w.intRotate(); err != nil {
		fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
		return nil
	}

	go func() {
		defer func() {
			if w.file != nil {
				fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
				w.file.Close()
			}
			w.stop <- true
		}()

		for {
			select {
			case <-w.stop:
				{
					goto EXIT
				}
			case <-w.rot:
				{
					if err := w.intRotate(); err != nil {
						fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
						continue
					}
				}
			case rec, ok := <-w.rec:
				{
					if !ok {
						return
					}
					now := time.Now()
					if (w.maxlines > 0 && w.maxlinesCurlines >= w.maxlines) ||
						(w.maxsize > 0 && w.maxsizeCursize >= w.maxsize) ||
						(w.daily && now.Day() != w.dailyOpendate) {
						if err := w.intRotate(); err != nil {
							fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
							continue
						}
					}

					// Perform the write
					n, err := fmt.Fprint(w.file, FormatLogRecord(w.format, rec))
					if err != nil {
						fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
						continue
					}

					// Update the counts
					w.maxlinesCurlines++
					w.maxsizeCursize += n
				}
			}
		}
	EXIT:
	}()

	return w
}

// Rotate Request that the logs rotate
func (w *FileLogWriter) Rotate() {
	w.rot <- true
}

// If this is called in a threaded context, it MUST be synchronized
func (w *FileLogWriter) intRotate() error {
	// Close any log file that may be open
	if w.file != nil {
		fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
		w.file.Close()
	}
	pieces := bytes.Split([]byte(w.filenameFormat), []byte{'%'})
	out := bytes.NewBuffer(make([]byte, 0, 64))
	for _, p := range pieces {
		if p[0] == 'D' {
			out.WriteString(time.Now().Format("2006-01-02"))
			if len(p) > 1 {
				out.Write(p[1:])
			}
		} else {
			out.Write(p)
		}
	}
	w.filename = out.String()
	// If we are keeping log files, move it to the next available number
	if w.rotate {
		_, err := os.Lstat(w.dir + w.filename)
		if err == nil { // file exists
			// Find the next available number
			num := 1
			fname := ""
			for ; err == nil && num <= 999; num++ {
				fname = w.filename + fmt.Sprintf(".%03d", num)
				_, err = os.Lstat(w.dir + fname)
			}
			// return error if the last file checked still existed
			if err == nil {
				return fmt.Errorf("Rotate: Cannot find free log number to rename %v%v", w.dir, w.filename)
			}

			// Rename the file to its newfound home
			err = os.Rename(w.dir+w.filename, w.dir+fname)
			if err != nil {
				return fmt.Errorf("Rotate: %v", err)
			}
		}
	}

	// Open the log file
	fd, err := os.OpenFile(w.dir+w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}

	w.file = fd

	now := time.Now()
	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	// Set the daily open date to the current date
	w.dailyOpendate = now.Day()

	// initialize rotation values
	w.maxlinesCurlines = 0
	w.maxsizeCursize = 0

	return nil
}

// SetFormat Set the logging format (chainable).  Must be called before the first log
// message is written.
func (w *FileLogWriter) SetFormat(format string) *FileLogWriter {
	w.format = format
	return w
}

// SetHeadFoot Set the logfile header and footer (chainable).  Must be called before the first log
// message is written.  These are formatted similar to the FormatLogRecord (e.g.
// you can use %D and %T in your header/footer for date and time).
func (w *FileLogWriter) SetHeadFoot(head, foot string) *FileLogWriter {
	w.header, w.trailer = head, foot
	if w.maxlinesCurlines == 0 {
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: time.Now()}))
	}
	return w
}

// SetRotateLines Set rotate at linecount (chainable). Must be called before the first log  message is written.
func (w *FileLogWriter) SetRotateLines(maxlines int) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateLines: %v\n", maxlines)
	w.maxlines = maxlines
	return w
}

// SetRotateSize Set rotate at size (chainable). Must be called before the first log message is written.
func (w *FileLogWriter) SetRotateSize(maxsize int) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateSize: %v\n", maxsize)
	w.maxsize = maxsize
	return w
}

// SetRotateDaily Set rotate daily (chainable). Must be called before the first log message is written.
func (w *FileLogWriter) SetRotateDaily(daily bool) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateDaily: %v\n", daily)
	w.daily = daily
	return w
}

// SetRotate changes whether or not the old logs are kept. (chainable) Must be
// called before the first log message is written.  If rotate is false, the
// files are overwritten; otherwise, they are rotated to another file before the new log is opened.
func (w *FileLogWriter) SetRotate(rotate bool) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotate: %v\n", rotate)
	w.rotate = rotate
	return w
}

// NewXMLLogWriter is a utility method for creating a FileLogWriter set up to
// output XML record log messages instead of line-based ones.
func NewXMLLogWriter(dir, fname string, rotate bool) *FileLogWriter {
	return NewFileLogWriter(dir, fname, rotate).SetFormat(
		`	<record level="%L">
		<timestamp>%D %T</timestamp>
		<source>%S</source>
		<message>%M</message>
	</record>`).SetHeadFoot("<log created=\"%D %T\">", "</log>")
}
