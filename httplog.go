package log4go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// RequestLogger struct
type RequestLogger struct {
	body     string
	datetime string
	url      string
	header   interface{}
}

// FlumeData 存储数据结构
type FlumeData map[string]interface{}

func (logger *RequestLogger) transRequest(writer *HTTPLogWriter) (*http.Request, error) {
	if writer != nil {
		var headers *map[string]interface{}
		if logger.header != nil {
			lheader := logger.header.(map[string]interface{})
			if lheader != nil {
				headers = &lheader
			}
		}
		if len(writer.headers) > 0 {
			if headers == nil {
				headers = &writer.headers
			} else {
				for key, value := range writer.headers {
					(*headers)[key] = value
				}
			}
		}
		if headers != nil {
			(*headers)["datetime"] = logger.datetime
		}
		data, err := json.Marshal([]FlumeData{
			FlumeData{
				"headers": headers,
				"body":    logger.body,
			},
		})
		if err != nil {
			return nil, err
		}
		buffer := new(bytes.Buffer)
		buffer.Write(data)
		url := writer.url
		if len(logger.url) > 0 {
			url = logger.url
		}
		req, err := http.NewRequest("POST", url, buffer)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
		return req, nil
	}
	return nil, nil
}

// LoggerProc log proc struct
type LoggerProc struct {
	loggers chan *RequestLogger //数据缓存
	stop    chan bool           //结束标志
	writer  *HTTPLogWriter      //日志输出
}

// NewLoggerProc 创建logger proc方法
func NewLoggerProc(writer *HTTPLogWriter, bufferSize int) *LoggerProc {
	proc := &LoggerProc{
		loggers: make(chan *RequestLogger, bufferSize),
		stop:    make(chan bool),
	}
	proc.writer = writer
	return proc
}

//启动日志协程
func (proc *LoggerProc) startLogger() {
	defer func() {
		proc.stop <- true
	}()
	for {
		select {
		case <-proc.stop:
			{
				goto EXIT
			}
		case log, ok := <-proc.loggers:
			{
				if !ok {
					return
				}
				proc.saveLogger(log)
			}
		}
	}
EXIT:
}

//停止日志协程
func (proc *LoggerProc) stopLogger() {
	proc.stop <- true
	<-proc.stop
	close(proc.loggers)
}

//处理日志
func (proc *LoggerProc) saveLogger(logger *RequestLogger) {
	if logger == nil || proc == nil {
		return
	}

	req, err := logger.transRequest(proc.writer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "trans request failed, err is %v", err)
		return
	}

	client := &http.Client{}
	client.Timeout = time.Duration(10) * time.Second
	response, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save log requst failed, api is %s, err is %v", proc.writer.url, err)
		return
	}

	response.Body.Close()
}

// HTTPLogWriter This log writer sends output to a http server
type HTTPLogWriter struct {
	procs   []*LoggerProc          //协程数组
	prand   *rand.Rand             //随机数
	url     string                 //上报链接
	headers map[string]interface{} //http headers
}

// 常量定义
const (
	LoggerProcCnt   = 2                           //默认处理日志的协程个数
	TimeFormateUnix = "2006-01-02T15:04:05+08:00" //unix format
)

// NewHTTPLogWriter 创建http writer
func NewHTTPLogWriter(url string, header map[string]interface{}, procSize int) *HTTPLogWriter {
	if procSize <= 0 {
		procSize = LoggerProcCnt
	}

	w := &HTTPLogWriter{
		procs:   make([]*LoggerProc, procSize),
		prand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		url:     url,
		headers: header,
	}

	for i := 0; i < procSize; i++ {
		w.procs[i] = NewLoggerProc(w, LogBufferLength)
	}

	for _, proc := range w.procs {
		if proc != nil {
			go proc.startLogger()
		}
	}
	return w
}

// SetURL 成员方法
func (w *HTTPLogWriter) SetURL(url string) {
	w.url = url
}

// GetURL 获取URL
func (w *HTTPLogWriter) GetURL() string {
	return w.url
}

// AddHeader 增加head
func (w *HTTPLogWriter) AddHeader(key, value string) {
	w.headers[key] = value
}

// GetHeaders 增加heads
func (w *HTTPLogWriter) GetHeaders() map[string]interface{} {
	return w.headers
}

// LogWrite This is the SocketLogWriter's output method
func (w *HTTPLogWriter) LogWrite(rec *LogRecord) {
	if rec != nil {
		url, body := "", ""
		var header interface{}
		if len(rec.Message) > 0 {
			body = rec.Message
		}
		if len(rec.Extend) > 0 {
			switch etp, edata := rec.GetExtend(); etp {
			case EXUrl:
				if len(edata) > 0 {
					url = edata[0].(string)
				}
			case EXUrlHead:
				if len(edata) > 1 {
					url = edata[0].(string)
					header = edata[1]
				}
			case EXUrlBody:
				if len(edata) > 1 {
					url = edata[0].(string)
					body = edata[1].(string)
				}
			case EXUrlHeadBody:
				if len(edata) > 2 {
					url = edata[0].(string)
					header = edata[1]
					body = edata[2].(string)
				}
			}
		}
		maxCnt := len(w.procs)
		index := w.prand.Intn(maxCnt)
		if index >= 0 && index < maxCnt {
			proc := w.procs[index]
			if proc != nil {
				proc.loggers <- &RequestLogger{
					body:     body,
					datetime: rec.Created.Format(TimeFormateUnix),
					url:      url,
					header:   header,
				}
			}
		}
	}

}

// Close 关闭
func (w *HTTPLogWriter) Close() {
	for index, proc := range w.procs {
		if proc != nil {
			proc.stopLogger()
			w.procs[index] = nil
		}
	}
}

// xmlToHTTPLogWriter xml创建http日志输出
func xmlToHTTPLogWriter(filename string, props []xmlProperty) (*HTTPLogWriter, bool) {
	url := ""
	headers := make(map[string]interface{})
	procnum := 0

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "url":
			url = strings.Trim(prop.Value, " \r\n")
		case "header":
			{
				strs := strings.Trim(prop.Value, " \r\n")
				if len(strs) > 0 {
					for _, tstr := range strings.Split(strs, ";") {
						ststrs := strings.Split(tstr, ":")
						if len(ststrs) >= 2 {
							headers[ststrs[0]] = ststrs[1]
						}
					}
				}
			}
		case "procnum":
			procnum, _ = strconv.Atoi(strings.Trim(prop.Value, " \r\n"))
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n", prop.Name, filename)
		}
	}

	// Check properties
	if len(url) == 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "url", filename)
		return nil, false
	}

	return NewHTTPLogWriter(url, headers, procnum), true
}
