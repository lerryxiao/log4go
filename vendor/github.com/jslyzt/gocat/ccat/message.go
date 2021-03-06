package ccat

import (
	"bytes"
	"time"
)

// 常量定义
const (
	SUCCESS = "0"
	FAIL    = "-1"
)

// Flush 输出
type Flush func(m Messager)

// MessageGetter 消息获取接口
type MessageGetter interface {
	GetData() *bytes.Buffer
	GetTime() time.Time
}

// Messager 消息接口
type Messager interface {
	MessageGetter
	AddData(k string, v ...string)
	SetStatus(status string)
	Complete()
}

// Message 消息结构
type Message struct {
	Type   string
	Name   string
	Status string

	timestampInNano int64

	data *bytes.Buffer

	flush Flush
}

// NewMessage 创建消息
func NewMessage(mtype, name string, flush Flush) *Message {
	return &Message{
		Type:            mtype,
		Name:            name,
		Status:          SUCCESS,
		timestampInNano: time.Now().UnixNano(),
		data:            new(bytes.Buffer),
		flush:           flush,
	}
}

// Complete 完成
func (m *Message) Complete() {
	m.flush(m)
}

// GetData 获取数据
func (m *Message) GetData() *bytes.Buffer {
	return m.data
}

// GetTime 获取时间
func (m *Message) GetTime() time.Time {
	return time.Unix(0, m.timestampInNano)
}

// SetTimestamp 设置时间戳
func (m *Message) SetTimestamp(timestampInNano int64) {
	m.timestampInNano = timestampInNano
}

// GetTimestamp 获取时间戳
func (m *Message) GetTimestamp() int64 {
	return m.timestampInNano
}

// AddData 增加数据
func (m *Message) AddData(k string, v ...string) {
	if m.data.Len() != 0 {
		m.data.WriteRune('&')
	}
	if len(v) == 0 {
		m.data.WriteString(k)
	} else {
		m.data.WriteString(k)
		m.data.WriteRune('=')
		m.data.WriteString(v[0])
	}
}

// SetStatus 设置状态
func (m *Message) SetStatus(status string) {
	m.Status = status
}
