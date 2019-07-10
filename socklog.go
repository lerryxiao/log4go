package log4go

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

// SocketLogWriter This log writer sends output to a socket
type SocketLogWriter struct {
	rec    chan *LogRecord
	stop   chan bool
	rptype uint8
}

// LogWrite This is the SocketLogWriter's output method
func (w *SocketLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

// Close 关闭
func (w *SocketLogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}

// SetReportType 设置上报类型
func (w *SocketLogWriter) SetReportType(tp uint8) {
	w.rptype = tp
}

// GetReportType 获取上报类型
func (w *SocketLogWriter) GetReportType() uint8 {
	return w.rptype
}

// NewSocketLogWriter 新建socket log writer
func NewSocketLogWriter(proto, hostport string) *SocketLogWriter {
	sock, err := net.Dial(proto, hostport)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewSocketLogWriter(%q): %s\n", hostport, err)
		return nil
	}

	w := &SocketLogWriter{
		rec:  make(chan *LogRecord, LogBufferLength),
		stop: make(chan bool),
	}

	go func() {
		defer func() {
			if sock != nil && proto == "tcp" {
				sock.Close()
			}
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
					// Marshall into JSON
					js, err := json.Marshal(rec)
					if err != nil {
						fmt.Fprintf(os.Stderr, "SocketLogWriter(%v): %v", hostport, err)
						return
					}
					_, err = sock.Write(js)
					if err != nil {
						fmt.Fprintf(os.Stderr, "SocketLogWriter(%v): %v", hostport, err)
						return
					}
				}
			}
		}
	EXIT:
	}()

	return w
}

// xmlToSocketLogWriter xml创建流日志输出
func xmlToSocketLogWriter(filename string, props []xmlProperty) (*SocketLogWriter, bool) {
	endpoint := ""
	protocol := "udp"

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "endpoint":
			endpoint = strings.Trim(prop.Value, " \r\n")
		case "protocol":
			protocol = strings.Trim(prop.Value, " \r\n")
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n", prop.Name, filename)
		}
	}

	// Check properties
	if len(endpoint) == 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "endpoint", filename)
		return nil, false
	}

	return NewSocketLogWriter(protocol, endpoint), true
}
