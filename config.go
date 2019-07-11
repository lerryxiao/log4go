package log4go

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type xmlProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type xmlFilter struct {
	Enabled  string        `xml:"enabled,attr"`
	Tag      string        `xml:"tag"`
	Level    string        `xml:"level"`
	Type     string        `xml:"type"`
	RptType  string        `xml:"report"`
	Property []xmlProperty `xml:"property"`
}

type xmlLoggerConfig struct {
	Filter []xmlFilter `xml:"filter"`
}

func getLevel(lvl string) level {
	switch lvl {
	case "FINEST", "finest":
		return FINEST
	case "FINE", "fine":
		return FINE
	case "DEBUG", "debug":
		return DEBUG
	case "TRACE", "trace":
		return TRACE
	case "INFO", "info":
		return INFO
	case "WARNING", "warning":
		return WARNING
	case "ERROR", "error":
		return ERROR
	case "FATAL", "fatal":
		return FATAL
	case "REPORT", "report":
		return REPORT
	default:
		return 0
	}
}

func getReportType(rptp string) uint8 {
	switch rptp {
	case "FLUME", "flume":
		return FLUME
	case "CAT", "cat":
		return CAT
	default:
		return 0
	}
}

// LoadConfiguration Load XML configuration; see examples/example.xml for documentation
func (log Logger) LoadConfiguration(filename string) {
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

	xc := new(xmlLoggerConfig)
	if err := xml.Unmarshal(contents, xc); err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not parse XML configuration in %q: %s\n", filename, err)
		os.Exit(1)
	}

	var (
		filt LogWriter
		lvl  level
		rptp uint8

		bad, good, enabled = false, true, false
	)

	for _, xmlfilt := range xc.Filter {
		bad, good, enabled = false, true, false
		if len(xmlfilt.Enabled) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required attribute %s for filter missing in %s\n", "enabled", filename)
			bad = true
		} else {
			enabled = (xmlfilt.Enabled != "false")
		}
		if !enabled {
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
		if lvl = getLevel(xmlfilt.Level); lvl == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter has unknown value in %s: %s\n", "level", filename, xmlfilt.Level)
			bad = true
		}
		rptp = getReportType(xmlfilt.RptType)
		if bad {
			os.Exit(1)
		}
		switch xmlfilt.Type {
		case "console":
			filt, good = xmlToConsoleLogWriter(filename, xmlfilt.Property)
		case "file":
			filt, good = xmlToFileLogWriter(filename, xmlfilt.Property)
		case "xml":
			filt, good = xmlToXMLLogWriter(filename, xmlfilt.Property)
		case "socket":
			filt, good = xmlToSocketLogWriter(filename, xmlfilt.Property)
		case "http":
			filt, good = xmlToHTTPLogWriter(filename, xmlfilt.Property)
		case "cat":
			filt, good = xmlToCatLogWriter(filename, xmlfilt.Property)
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not load XML configuration in %s: unknown filter type \"%s\"\n", filename, xmlfilt.Type)
			os.Exit(1)
		}
		if !good {
			os.Exit(1)
		}
		filt.SetReportType(rptp)
		log.AddFilter(xmlfilt.Tag, filt, lvl)
	}
}
