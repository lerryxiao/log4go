package ccat

// Heartbeat 心跳
type Heartbeat struct {
	Message
}

// Complete 处理
func (e *Heartbeat) Complete() {
	e.Message.flush(e)
}
