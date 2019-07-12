package log4go

import (
	"fmt"
	"github.com/lerryxiao/log4go/log/define"
	"github.com/lerryxiao/log4go/log"
	"os"
	"io/ioutil"
	"encoding/xml"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////////

// Version information
const (
	L4GVersion = "log4go-v3.0.1"
	L4GMajor   = 3
	L4GMinor   = 0
	L4GBuild   = 1
)

////////////////////////////////////////////////////////////////////////////////////

var (
	createFuns = map[string]define.WriterCreater{
		"console": log.XMLToConsoleLogWriter,
		"file":    log.XMLToFileLogWriter,
		"xml":     log.XMLToXMLLogWriter,
		"socket":  log.XMLToSocketLogWriter,
		"http":    log.XMLToHTTPLogWriter,
	}
)

////////////////////////////////////////////////////////////////////////////////////

// NewLogger 创建
func NewLogger() Logger {
	return make(Logger)
}

// RegistCreater 注册创建者
func RegistCreater(key string, fun define.WriterCreater) {
	if val, ok := createFuns[key]; ok == false || val == nil {
		createFuns[key] = fun
	}
}

// LoadConfiguration Load XML configuration; see examples/example.xml for documentation
func LoadConfiguration(filename string, log define.Logger) {
	log.Close()

	// Open the configuration file
	fd, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not open %q for reading: %s\n", filename, err)
		os.Exit(1)
	}

	contents, err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not read %q: %s\n", filename, err)
		os.Exit(1)
	}

	xc := new(define.XMLLoggerConfig)
	if err := xml.Unmarshal(contents, xc); err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not parse XML configuration in %q: %s\n", filename, err)
		os.Exit(1)
	}

	var (
		lvl          uint8
		bad, enabled bool
	)

	for _, xmlfilt := range xc.Filter {
		bad, enabled = false, false
		if len(xmlfilt.Enabled) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required attribute %s for filter missing in %s\n", "enabled", filename)
			bad = true
		} else if strings.ToLower(xmlfilt.Enabled) == "true" || xmlfilt.Enabled == "1" {
			enabled = true
		}
		if enabled == false {
			continue
		}
		if len(xmlfilt.Tag) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "tag", filename)
			bad = true
		}
		if len(xmlfilt.Type) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "type", filename)
			bad = true
		}
		if len(xmlfilt.Level) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "level", filename)
			bad = true
		}
		if lvl = define.GetLevel(xmlfilt.Level); lvl == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter has unknown value in %s: %s\n", "level", filename, xmlfilt.Level)
			bad = true
		}
		fun, ok := createFuns[xmlfilt.Type]
		if fun == nil || ok == false {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not load XML configuration in %s: unknown filter type \"%s\"\n", filename, xmlfilt.Type)
			bad = true
		}
		if bad {
			os.Exit(1)
		}
		filt, good := fun(filename, xmlfilt.Property)
		if good == false || filt == nil {
			os.Exit(1)
		}
		filt.SetReportType(define.GetReportType(xmlfilt.RptType))
		log.AddFilter(xmlfilt.Tag, filt, lvl)
	}
}
