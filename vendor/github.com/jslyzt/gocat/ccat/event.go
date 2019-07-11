package ccat

// Event 事件定义
type Event struct {
	Message
}

// Complete 处理
func (e *Event) Complete() {
	e.Message.flush(e)
}
