package log

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/lerryxiao/log4go/log/define"
)

// FileLogWriter 文件日志输出
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

	rptype uint8
}

// LogWrite 输出方法
func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

// Close 关闭
func (w *FileLogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}

// SetReportType 设置上报类型
func (w *FileLogWriter) SetReportType(tp uint8) {
	w.rptype = tp
}

// GetReportType 获取上报类型
func (w *FileLogWriter) GetReportType() uint8 {
	return w.rptype
}

// NewFileLogWriter 创建文件输出节点
func NewFileLogWriter(dir, fname string, rotate bool) *FileLogWriter {
	w := &FileLogWriter{
		rec:            make(chan *LogRecord, define.LogBufferLength),
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

// Rotate 设置转存
func (w *FileLogWriter) Rotate() {
	w.rot <- true
}

// 开始转存
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

// SetFormat 设置输出格式
func (w *FileLogWriter) SetFormat(format string) *FileLogWriter {
	w.format = format
	return w
}

// SetHeadFoot 设置文件头信息
func (w *FileLogWriter) SetHeadFoot(head, foot string) *FileLogWriter {
	w.header, w.trailer = head, foot
	if w.maxlinesCurlines == 0 {
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: time.Now()}))
	}
	return w
}

// SetRotateLines 设置转存行数量阀值
func (w *FileLogWriter) SetRotateLines(maxlines int) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateLines: %v\n", maxlines)
	w.maxlines = maxlines
	return w
}

// SetRotateSize 设置转存大小阀值
func (w *FileLogWriter) SetRotateSize(maxsize int) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateSize: %v\n", maxsize)
	w.maxsize = maxsize
	return w
}

// SetRotateDaily 设置每天转存
func (w *FileLogWriter) SetRotateDaily(daily bool) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateDaily: %v\n", daily)
	w.daily = daily
	return w
}

// SetRotate 设置已转存
func (w *FileLogWriter) SetRotate(rotate bool) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotate: %v\n", rotate)
	w.rotate = rotate
	return w
}

// Parse a number with K/M/G suffixes based on thousands (1000) or 2^10 (1024)
func strToNumSuffix(str string, mult int) int {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, _ := strconv.Atoi(str)
	return parsed * num
}

// XMLToFileLogWriter xml创建文件日志输出
func XMLToFileLogWriter(filename string, props []define.XMLProperty) (LogWriter, bool) {
	file := ""
	format := "[%D %T] [%L] (%S) %M"
	maxlines := 0
	maxsize := 0
	daily := false
	rotate := false
	dir := ""

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
		case "format":
			format = strings.Trim(prop.Value, " \r\n")
		case "maxlines":
			maxlines = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024)
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "dir":
			dir = strings.Trim(prop.Value, " \r\n")
			if !strings.HasSuffix(dir, "/") {
				dir += "/"
			}
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n", prop.Name, filename)
		}
	}

	// Check properties
	if len(file) == 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "filename", filename)
		return nil, false
	}

	flw := NewFileLogWriter(dir, file, rotate)
	flw.SetFormat(format)
	flw.SetRotateLines(maxlines)
	flw.SetRotateSize(maxsize)
	flw.SetRotateDaily(daily)
	return flw, true
}

// NewXMLLogWriter 创建xml日志输出
func NewXMLLogWriter(dir, fname string, rotate bool) *FileLogWriter {
	return NewFileLogWriter(dir, fname, rotate).SetFormat(
		`	<record level="%L">
		<timestamp>%D %T</timestamp>
		<source>%S</source>
		<message>%M</message>
	</record>`).SetHeadFoot("<log created=\"%D %T\">", "</log>")
}

// XMLToXMLLogWriter xml创建xml日志输出
func XMLToXMLLogWriter(filename string, props []define.XMLProperty) (LogWriter, bool) {
	file := ""
	maxrecords := 0
	maxsize := 0
	daily := false
	rotate := false
	dir := ""

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
		case "maxrecords":
			maxrecords = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024)
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "dir":
			dir = strings.Trim(prop.Value, " \r\n")
			if !strings.HasSuffix(dir, "/") {
				dir += "/"
			}
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Warning: Unknown property \"%s\" for xml filter in %s\n", prop.Name, filename)
		}
	}

	// Check properties
	if len(file) == 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for xml filter missing in %s\n", "filename", filename)
		return nil, false
	}

	xlw := NewXMLLogWriter(dir, file, rotate)
	xlw.SetRotateLines(maxrecords)
	xlw.SetRotateSize(maxsize)
	xlw.SetRotateDaily(daily)
	return xlw, true
}
