package define

// XMLProperty property属性
type XMLProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// XMLFilter filter过滤器
type XMLFilter struct {
	Enabled  string        `xml:"enabled,attr"`
	Tag      string        `xml:"tag"`
	Level    string        `xml:"level"`
	Type     string        `xml:"type"`
	RptType  string        `xml:"report"`
	Property []XMLProperty `xml:"property"`
}

// XMLLoggerConfig logger配置
type XMLLoggerConfig struct {
	Filter []XMLFilter `xml:"filter"`
}

// GetLevel 等级
func GetLevel(lvl string) uint8 {
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

// GetReportType 上报类型
func GetReportType(rptp string) uint8 {
	switch rptp {
	case "FLUME", "flume":
		return FLUME
	case "CAT", "cat":
		return CAT
	default:
		return 0
	}
}
