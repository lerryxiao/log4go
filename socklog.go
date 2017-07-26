// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// This log writer sends output to a socket
type SocketLogWriter struct {
	rec  chan *LogRecord
	stop chan bool
}

// This is the SocketLogWriter's output method
func (w *SocketLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *SocketLogWriter) Close() {
	close(w.rec)
	<-w.stop
}

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

		for rec := range w.rec {
			// Marshall into JSON
			js, err := json.Marshal(rec)
			if err != nil {
				fmt.Fprint(os.Stderr, "SocketLogWriter(%q): %s", hostport, err)
				return
			}

			_, err = sock.Write(js)
			if err != nil {
				fmt.Fprint(os.Stderr, "SocketLogWriter(%q): %s", hostport, err)
				return
			}
		}
	}()

	return w
}
