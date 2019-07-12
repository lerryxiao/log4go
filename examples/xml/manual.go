package main

import (
	l4g "github.com/lerryxiao/log4go"
)

func main() {
	log := l4g.NewLogger()
	l4g.LoadConfiguration("example.xml", log)

	log.Finest("This will only go to those of you really cool UDP kids!  If you change enabled=true.")
	log.Debug("Oh no!  %d + %d = %d!", 2, 2, 2+2)
	log.Info("About that time, eh chaps?")
}
