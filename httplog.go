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
	body     string
	datetime string
	url      string
	header   interface{}
}

type FlumeData map[string]interface{}

func (logger *RequestLogger) transRequest(writer *HttpLogWriter) (*http.Request, error) {
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

//log proc struct
type loggerProc struct {
	loggers chan *RequestLogger //数据缓存
	stop    chan bool           //结束标志
	writer  *HttpLogWriter      //日志输出
}

func NewLoggerProc(writer *HttpLogWriter, bufferSize int) *loggerProc {
	proc := &loggerProc{
		loggers: make(chan *RequestLogger, bufferSize),
		stop:    make(chan bool),
	}
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

	client := &http.Client{}
	client.Timeout = time.Duration(10) * time.Second
	response, err := client.Do(req)
	if err != nil {
		fmt.Fprint(os.Stderr, "save log requst failed, api is %s, err is %v", proc.writer.url, err)
		return
	}

	response.Body.Close()
}

// This log writer sends output to a http server
type HttpLogWriter struct {
	procs   []*loggerProc          //协程数组
	prand   *rand.Rand             //随机数
	url     string                 //上报链接
	headers map[string]interface{} //http headers
}

const (
	LOGGER_PROC_CNT   = 2                           //默认处理日志的协程个数
	TIME_FORMATE_UNIX = "2006-01-02T15:04:05+08:00" //unix format
)

func NewHttpLogWriter(url string, header map[string]interface{}, procSize int) *HttpLogWriter {
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

func (w *HttpLogWriter) GetHeaders() map[string]interface{} {
	return w.headers
}

// This is the SocketLogWriter's output method
func (w *HttpLogWriter) LogWrite(rec *LogRecord) {
	if rec != nil {
		url, body := "", ""
		var header interface{}
		if len(rec.Message) > 0 {
			body = rec.Message
		}
		if len(rec.Extend) > 0 {
			switch etp, edata := rec.GetExtend(); etp {
			case EX_URL:
				if len(edata) > 0 {
					url = edata[0].(string)
				}
			case EX_URL_HEAD:
				if len(edata) > 1 {
					url = edata[0].(string)
					header = edata[1]
				}
			case EX_URL_BODY:
				if len(edata) > 1 {
					url = edata[0].(string)
					body = edata[1].(string)
				}
			case EX_URL_HEAD_BODY:
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
