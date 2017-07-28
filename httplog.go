package log4go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

//Requst logger struct
type RequestLogger struct {
	Headers  *map[string]string `json:"headers"`
	Body     string             `json:"body"`
	datetime string
}

func (logger *RequestLogger) transRequest(writer *HttpLogWriter) (*http.Request, error) {
	if writer != nil {
		writer.headers["datetime"] = logger.datetime
		logger.Headers = &writer.headers
		data, err := json.Marshal(logger)
		if err != nil {
			return nil, err
		}

		buffer := new(bytes.Buffer)
		buffer.Write(data)

		req, err := http.NewRequest("POST", writer.url, buffer)
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

// This is the SocketLogWriter's output method
func (w *HttpLogWriter) LogWrite(rec *LogRecord) {
	if rec != nil {
		bodyInfo, err := json.Marshal(map[string]string{"data": rec.Message})
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
					Body:     string(bodyInfo),
					datetime: rec.Created.Format(TIME_FORMATE_UNIX),
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
