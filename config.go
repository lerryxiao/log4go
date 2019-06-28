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
	Property []xmlProperty `xml:"property"`
}

type xmlLoggerConfig struct {
	Filter []xmlFilter `xml:"filter"`
}

func getLevel(lvl string) level {
	switch lvl {
	case "FINEST":
		return FINEST
	case "FINE":
		return FINE
	case "DEBUG":
		return DEBUG
	case "TRACE":
		return TRACE
	case "INFO":
		return INFO
	case "WARNING":
		return WARNING
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	case "REPORT":
		return REPORT
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
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not load XML configuration in %s: unknown filter type \"%s\"\n", filename, xmlfilt.Type)
			os.Exit(1)
		}
		if !good {
			os.Exit(1)
		}
		log[xmlfilt.Tag] = &Filter{lvl, filt}
	}
}
