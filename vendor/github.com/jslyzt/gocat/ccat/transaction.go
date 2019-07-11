package ccat

import (
	"time"
)

// Transaction 事务
type Transaction struct {
	Message
	durationInNano      int64
	durationStartInNano int64
}

// NewTransaction 创建事务
func NewTransaction(mtype, name string, flush Flush) *Transaction {
	return &Transaction{
		Message:             *NewMessage(mtype, name, flush),
		durationStartInNano: time.Now().UnixNano(),
	}
}

// Complete 完成
func (t *Transaction) Complete() {
	if t.durationInNano == 0 {
		durationNano := time.Now().UnixNano() - t.durationStartInNano
		t.durationInNano = durationNano
	}
	t.Message.flush(t)
}

// GetDuration 获取时间
func (t *Transaction) GetDuration() int64 {
	return t.durationInNano
}

// SetDuration 设置时间
func (t *Transaction) SetDuration(durationInNano int64) {
	t.durationInNano = durationInNano
}

// SetDurationStart 设置开始时间
func (t *Transaction) SetDurationStart(durationStartInNano int64) {
	t.durationStartInNano = durationStartInNano
}
