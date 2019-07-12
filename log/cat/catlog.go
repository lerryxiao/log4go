package cat

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/jslyzt/gocat/ccat"
	"github.com/jslyzt/gocat/gcat"
	"github.com/lerryxiao/log4go/log/define"
	"github.com/spf13/cast"
)

////////////////////////////////////////////////////////////////////////////////////////////

// 变量定义
var (
	cat       = gcat.Instance()
	catDomain = ""
	LogKey    = "cat"
)

func initDomain(domain string) {
	if len(catDomain) > 0 {
		if len(domain) <= 0 || domain == catDomain {
			return
		}
		fmt.Fprintf(os.Stderr, "cat has init domain: %v, should not init: %v", catDomain, domain)
		return
	}
	catDomain = domain
	gcat.Init(domain, gcat.DefaultConfigForCat2())
}

////////////////////////////////////////////////////////////////////////////////////////////

// LogWriter This log writer sends output to cat
type LogWriter struct {
	rec      chan *define.LogRecord
	stop     chan bool
	rptype   uint8
	rptgroup string
}

// LogWrite This is the SocketLogWriter's output method
func (w *LogWriter) LogWrite(rec *define.LogRecord) {
	w.rec <- rec
}

// Close 关闭
func (w *LogWriter) Close() {
	w.stop <- true
	<-w.stop
	close(w.rec)
}

// SetReportType 设置上报类型
func (w *LogWriter) SetReportType(tp uint8) {
	w.rptype = tp
}

// GetReportType 获取上报类型
func (w *LogWriter) GetReportType() uint8 {
	return w.rptype
}

func (w *LogWriter) getArg(args []interface{}, index int) interface{} {
	if index >= len(args) {
		return nil
	}
	return args[index]
}

// NewCatLogWriter 新建socket log writer
func NewCatLogWriter(domain, group string) *LogWriter {
	if len(domain) > 0 {
		initDomain(domain)
	}
	if len(catDomain) <= 0 {
		fmt.Fprintf(os.Stderr, "NewLogWriter(%v) domain is nil", domain)
		return nil
	}

	w := &LogWriter{
		rec:      make(chan *define.LogRecord, define.LogBufferLength),
		stop:     make(chan bool),
		rptgroup: group,
	}

	go func() {
		defer func() {
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
					if rec == nil || len(rec.Extend) <= 0 {
						continue
					}
					tp, data := rec.GetExtend()
					switch tp {
					case define.EXCatTransaction:
						go w.dealTransaction(data)
					case define.EXCatEvent:
						go w.dealEvent(data)
					case define.EXCatError:
						go w.dealError(data)
					case define.EXCatMetricCount:
						go w.dealMetricCount(data)
					case define.EXCatMetricDuration:
						go w.dealMetricDuration(data)
					}
				}
			}
		}
	EXIT:
	}()

	return w
}

func (w *LogWriter) getName(v interface{}) string {
	if v == nil {
		return ""
	}
	switch vl := v.(type) {
	case string:
		return vl
	case func() string:
		return vl()
	case map[interface{}]interface{}:
		name := ""
		for k, v := range vl {
			name = name + cast.ToString(k) + ":" + cast.ToString(v) + "_"
		}
		if len(name) > 0 {
			name = name[:len(name)-1]
		}
		return name
	case []interface{}:
		name := ""
		for _, v := range vl {
			name = name + cast.ToString(v) + "_"
		}
		if len(name) > 0 {
			name = name[:len(name)-1]
		}
		return name
	default:
		fmt.Fprintf(os.Stderr, "unsupport name type: %v, value: %v", reflect.TypeOf(v), v)
	}
	return ""
}

func (w *LogWriter) addMsgData(m *ccat.Message, v interface{}) {
	if m != nil && v != nil {
		switch vl := v.(type) {
		case map[string]interface{}:
			{
				for key, val := range vl {
					m.AddData(key, cast.ToString(val))
				}
			}
		case []interface{}:
			{
				if len(vl)%2 == 0 {
					for i := 0; i < len(vl)-1; i = i + 2 {
						m.AddData(cast.ToString(vl[i]), cast.ToString(vl[i+1]))
					}
				} else {
					for id, val := range vl {
						m.AddData(cast.ToString(id), cast.ToString(val))
					}
				}
			}
		default:
			fmt.Fprintf(os.Stderr, "unsupport addMsgData type: %v, value: %v", reflect.TypeOf(v), v)
		}
	}
}

func (w *LogWriter) setMsgStatus(m *ccat.Message, v interface{}) {
	if m != nil && v != nil {
		switch vl := v.(type) {
		case string:
			{
				if vl == "ok" || vl == "OK" {
					m.SetStatus(gcat.SUCCESS)
				} else {
					m.SetStatus(gcat.FAIL)
					m.AddData("err", vl)
				}
			}
		case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
			{
				val := cast.ToInt(v)
				if val > 0 {
					m.SetStatus(gcat.SUCCESS)
				} else {
					m.SetStatus(gcat.FAIL)
					m.AddData("errcode", cast.ToString(val))
				}
			}
		case bool:
			{
				if vl == true {
					m.SetStatus(gcat.SUCCESS)
				} else {
					m.SetStatus(gcat.FAIL)
				}
			}
		default:
			fmt.Fprintf(os.Stderr, "unsupport setMsgStatus type: %v, value: %v", reflect.TypeOf(v), v)
		}
	}
}

func (w *LogWriter) dealTransaction(data []interface{}) {
	dtl := len(data)
	if dtl > 0 {
		t := cat.NewTransaction(w.rptgroup, w.getName(w.getArg(data, 0)))
		if dtl > 1 {
			w.setMsgStatus(&t.Message, w.getArg(data, 1))
		}
		if dtl > 2 {
			w.setMsgStatus(&t.Message, w.getArg(data, 2))
		}
		t.Complete()
	}
}

func (w *LogWriter) dealEvent(data []interface{}) {
	dtl := len(data)
	if dtl > 0 {
		t := cat.NewEvent(w.rptgroup, w.getName(w.getArg(data, 0)))
		if dtl > 1 {
			w.setMsgStatus(&t.Message, w.getArg(data, 1))
		}
		if dtl > 2 {
			w.addMsgData(&t.Message, w.getArg(data, 2))
		}
		t.Complete()
	}
}

func (w *LogWriter) dealError(data []interface{}) {
	dtl := len(data)
	if dtl > 0 {
		var category = w.getName(w.getArg(data, 0))
		if len(category) <= 0 {
			category = "error"
		}
		t := cat.NewEvent(w.rptgroup+"_error", category)
		t.SetStatus(ccat.FAIL)
		if dtl > 1 {
			t.AddData(gcat.NewStackTrace(1, cast.ToString(w.getArg(data, 1))).String())
		} else {
			t.AddData(gcat.NewStackTrace(1, "").String())
		}
		t.Complete()
	}
}

func (w *LogWriter) dealMetricCount(data []interface{}) {
	dtl := len(data)
	if dtl > 0 {
		if dtl > 1 {
			cat.LogMetricForCount(w.rptgroup+"_"+w.getName(w.getArg(data, 0)), cast.ToInt(w.getArg(data, 1)))
		} else {
			cat.LogMetricForCount(w.rptgroup + "_" + w.getName(w.getArg(data, 0)))
		}
	}
}

func (w *LogWriter) dealMetricDuration(data []interface{}) {
	dtl := len(data)
	if dtl > 1 {
		cat.LogMetricForDuration(w.rptgroup+"_"+w.getName(w.getArg(data, 0)), cast.ToInt64(w.getArg(data, 1)))
	}
}

////////////////////////////////////////////////////////////////////////////////////////////

// XMLToCatLogWriter xml创建cat日志输出
func XMLToCatLogWriter(filename string, props []define.XMLProperty) (define.LogWriter, bool) {
	var (
		domain, group string
	)

	// Parse properties
	for _, prop := range props {
		switch prop.Name {
		case "domain":
			domain = strings.Trim(prop.Value, " \r\n")
		case "group":
			group = strings.Trim(prop.Value, " \r\n")
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Warning: Unknown property \"%s\" for file filter in %s\n", prop.Name, filename)
		}
	}

	// Check properties
	if len(domain) <= 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "domain", filename)
		return nil, false
	}
	if len(group) <= 0 {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "group", filename)
		return nil, false
	}

	return NewCatLogWriter(domain, group), true
}
