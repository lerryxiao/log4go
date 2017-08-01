package log4go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"time"
)

//Requst logger struct
type RequestLogger struct {
	body     string
	datetime string
	url      string
	header   string
}

func (logger *RequestLogger) transRequest(writer *HttpLogWriter) (*http.Request, error) {
	if writer != nil {
		header := ""
		if len(logger.header) <= 0 {
			writer.headers["datetime"] = logger.datetime
			data, err := json.Marshal(writer.headers)
			if err != nil {
				return nil, err
			}
			header = string(data)
		} else {
			header = logger.header
		}
		data, err := json.Marshal(map[string]string{
			"headers": header,
			"body":    logger.body,
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

//log proc struct
type loggerProc struct {
	loggers chan *RequestLogger //数据缓存
	stop    chan bool           //结束标志
	client  *http.Client        //http客户端
	writer  *HttpLogWriter      //日志输出
}

func NewLoggerProc(writer *HttpLogWriter, bufferSize int) *loggerProc {
	proc := &loggerProc{
		loggers: make(chan *RequestLogger, bufferSize),
		stop:    make(chan bool),
		client:  &http.Client{},
	}
	proc.client.Timeout = time.Duration(10) * time.Second
	proc.writer = writer
	return proc
}

//启动日志协程
func (proc *loggerProc) startLogger() {
	defer func() {
		proc.stop <- true
	}()
	for {
		select {
		case log, ok := <-proc.loggers:
			if !ok {
				return
			}
			proc.saveLogger(log)
		}
	}
}

//停止日志协程
func (proc *loggerProc) stopLogger() {
	close(proc.loggers)
	<-proc.stop
}

//处理日志
func (proc *loggerProc) saveLogger(logger *RequestLogger) {
	if logger == nil || proc == nil {
		return
	}

	req, err := logger.transRequest(proc.writer)
	if err != nil {
		fmt.Fprint(os.Stderr, "trans request failed, err is %v", err)
		return
	}

	response, err := proc.client.Do(req)
	if err != nil {
		fmt.Fprint(os.Stderr, "save log requst failed, api is %s, err is %v", proc.writer.url, err)
		return
	}

	response.Body.Close()
}

// This log writer sends output to a http server
type HttpLogWriter struct {
	procs   []*loggerProc     //协程数组
	prand   *rand.Rand        //随机数
	url     string            //上报链接
	headers map[string]string //http headers
}

const (
	LOGGER_PROC_CNT   = 2                           //默认处理日志的协程个数
	TIME_FORMATE_UNIX = "2006-01-02T15:04:05+08:00" //unix format
)

func NewHttpLogWriter(url string, header map[string]string, procSize int) *HttpLogWriter {
	if procSize <= 0 {
		procSize = LOGGER_PROC_CNT
	}

	w := &HttpLogWriter{
		procs:   make([]*loggerProc, procSize),
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

//成员方法
func (w *HttpLogWriter) SetUrl(url string) {
	w.url = url
}

func (w *HttpLogWriter) GetUrl() string {
	return w.url
}

func (w *HttpLogWriter) AddHeader(key, value string) {
	w.headers[key] = value
}

func (w *HttpLogWriter) GetHeaders() map[string]string {
	return w.headers
}

func any2string(any interface{}) string {
	switch any.(type) {
	case string:
		return any.(string)
	default:
		data, err := json.Marshal(any)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

func getExtendReport(extends []interface{}) (bool, []string) {
	extsz := len(extends)
	if extsz > 0 {
		switch extends[0].(type) {
		case int8, int16, int32, int64, int:
			if reflect.ValueOf(extends[0]).Int() == int64(EX_REPORT) {
				rtn := make([]string, 0)
				for i := 1; i < extsz; i++ {
					rtn = append(rtn, any2string(extends[i]))
				}
				return true, rtn
			}
		}
	}
	return false, nil
}

// This is the SocketLogWriter's output method
func (w *HttpLogWriter) LogWrite(rec *LogRecord) {
	if rec != nil {
		url, header, body := "", "", ""
		if len(rec.Message) > 0 {
			body = rec.Message
		} else if len(rec.Extend) > 0 {
			succ, data := getExtendReport(rec.Extend)
			if succ == true {
				dlen := len(data)
				if dlen > 0 {
					url = data[0]
				}
				if dlen > 1 {
					header = data[1]
				}
				if dlen > 2 {
					body = data[2]
				}
			}
		}
		bodyInfo, err := json.Marshal(map[string]string{"data": body})
		if err != nil {
			fmt.Fprint(os.Stderr, "HttpLogWriter LogWrite json Marshal failed, err is %v", err)
			return
		}
		maxCnt := len(w.procs)
		index := w.prand.Intn(maxCnt)
		if index >= 0 && index < maxCnt {
			proc := w.procs[index]
			if proc != nil {
				proc.loggers <- &RequestLogger{
					body:     string(bodyInfo),
					datetime: rec.Created.Format(TIME_FORMATE_UNIX),
					url:      url,
					header:   header,
				}
			}
		}
	}

}

func (w *HttpLogWriter) Close() {
	for index, proc := range w.procs {
		if proc != nil {
			proc.stopLogger()
			w.procs[index] = nil
		}
	}
}
