package main

import (
	"time"
	l4g "github.com/lerryxiao/log4go"
)

func main() {
	l := l4g.NewLogger()
	l.AddFilter("stdout", l4g.NewConsoleLogWriter(), l4g.DEBUG)
	l.Info("The time is now: %s", time.Now().Format("15:04:05 MST 2006/01/02"))
	l.Close()
}
