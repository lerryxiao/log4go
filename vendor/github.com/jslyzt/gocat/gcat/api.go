package gcat

import (
	"time"

	"github.com/jslyzt/gocat/ccat"
)

// CatInstance cat实例
type CatInstance struct {
}

// Instance 实例
func Instance() *CatInstance {
	return &CatInstance{}
}

// flush 输出
func (t *CatInstance) flush(m ccat.Messager) {
	ccat.Send(m)
}

// NewTransaction 新建事务
func (t *CatInstance) NewTransaction(mtype, name string) *ccat.Transaction {
	return ccat.NewTransaction(mtype, name, t.flush)
}

// NewCompletedTransactionWithDuration 新建超时完成事务
func (t *CatInstance) NewCompletedTransactionWithDuration(mtype, name string, durationInNano int64) {
	var trans = t.NewTransaction(mtype, name)
	trans.SetDuration(durationInNano)
	if durationInNano > 0 && durationInNano < 60*time.Second.Nanoseconds() {
		trans.SetTimestamp(time.Now().UnixNano() - durationInNano)
	}
	trans.SetStatus(ccat.SUCCESS)
	trans.Complete()
}

// NewEvent 新建事件
func (t *CatInstance) NewEvent(mtype, name string) *ccat.Event {
	return &ccat.Event{
		Message: *ccat.NewMessage(mtype, name, t.flush),
	}
}

// NewHeartbeat 新建心跳
func (t *CatInstance) NewHeartbeat(mtype, name string) *ccat.Heartbeat {
	return &ccat.Heartbeat{
		Message: *ccat.NewMessage(mtype, name, t.flush),
	}
}

// LogEvent 日志事件
func (t *CatInstance) LogEvent(mtype, name string, args ...string) {
	var e = t.NewEvent(mtype, name)
	if len(args) > 0 {
		e.SetStatus(args[0])
	}
	if len(args) > 1 {
		e.AddData(args[1])
	}
	e.Complete()
}

// LogError 日志错误
func (t *CatInstance) LogError(err error, args ...string) {
	var category = "error"
	if len(args) > 0 {
		category = args[0]
	}
	var e = t.NewEvent("Exception", category)
	var buf = NewStackTrace(2, err.Error())
	e.SetStatus(ccat.FAIL)
	e.AddData(buf.String())
	e.Complete()
}

// LogMetricForCount 日志调节
func (t *CatInstance) LogMetricForCount(mname string, args ...int) {
	if len(args) == 0 {
		ccat.LogMetricForCount(mname, 1)
	} else {
		ccat.LogMetricForCount(mname, args[0])
	}
}

// LogMetricForDuration 日志条件
func (t *CatInstance) LogMetricForDuration(mname string, duration int64) {
	ccat.LogMetricForDuration(mname, duration)
}
