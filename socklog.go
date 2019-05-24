package log4go

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// SocketLogWriter This log writer sends output to a socket
type SocketLogWriter struct {
	rec  chan *LogRecord
	stop chan bool
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
