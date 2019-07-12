package ccat

/*
#cgo darwin LDFLAGS: -L${SRCDIR} -lcatclient_darwin
#cgo windows LDFLAGS: -L${SRCDIR} -lcatclient_win -lm -lpthread -lwsock32 -lws2_32
#cgo linux LDFLAGS: -L${SRCDIR} -lcatclient_linux -lm -lpthread

#include <stdlib.h>
#include "ccat.h"
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"
)

var (
	ch = make(chan interface{}, 128)
	wg sync.WaitGroup
)

// Init 初始化
func Init(domain string) {
	var cdomain = C.CString(domain)
	defer C.free(unsafe.Pointer(cdomain))
	C.catClientInit(cdomain)
}

// BuildConfig 创建配置
func BuildConfig( encoderType, enableHeartbeat,	enableSampling,	enableDebugLog int,) C.CatClientConfig {
	return C.CatClientConfig{
		C.int(encoderType),
		C.int(enableHeartbeat),
		C.int(enableSampling),
		0,
		C.int(enableDebugLog),
	}
}

// InitWithConfig 使用配置文件初始化
func InitWithConfig(domain string,  _config C.CatClientConfig) {
	var cdomain = C.CString(domain)
	defer C.free(unsafe.Pointer(cdomain))
	C.catClientInitWithConfig(cdomain, &_config)
}

// Background 工作线程
func Background() {
	// We need running ccat functions on the same thread due to ccat is using a thread local.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	wg.Add(1)
	defer wg.Done()

	for m := range ch {
		switch m := m.(type) {
		case *Transaction:
			LogTransaction(m)
		case *Event:
			LogEvent(m)
		}
	}
}

// Shutdown 关闭
func Shutdown() {
	close(ch)
}

// Wait 等待
func Wait() {
	wg.Wait()
}

// ShutdownAndWait 等待关闭
func ShutdownAndWait() {
	Shutdown()
	Wait()
}

// Send 发送数据
func Send(m Messager) {
	ch <- m
}

// LogTransaction 日志处理
func LogTransaction(trans *Transaction) {
	var (
		ctype   = C.CString(trans.Type)
		cname   = C.CString(trans.Name)
		cstatus = C.CString(trans.Status)
		cdata   = C.CString(trans.GetData().String())
	)
	defer func() {
		C.free(unsafe.Pointer(ctype))
		C.free(unsafe.Pointer(cname))
		C.free(unsafe.Pointer(cstatus))
		C.free(unsafe.Pointer(cdata))
	}() 
	C.callLogTransaction(
		ctype, cname, cstatus, cdata,
		C.ulonglong(trans.GetTimestamp()/1000/1000),
		C.ulonglong(trans.GetTimestamp()/1000/1000),
		C.ulonglong(trans.GetDuration()/1000/1000),
	)
}

// LogEvent 日志事件
func LogEvent(event *Event) {
	var (
		ctype   = C.CString(event.Type)
		cname   = C.CString(event.Name)
		cstatus = C.CString(event.Status)
		cdata   = C.CString(event.GetData().String())
	)
	defer func() {
		C.free(unsafe.Pointer(ctype))
		C.free(unsafe.Pointer(cname))
		C.free(unsafe.Pointer(cstatus))
		C.free(unsafe.Pointer(cdata))
	}() 
	C.callLogEvent(
		ctype, cname, cstatus, cdata,
		C.ulonglong(event.GetTimestamp()/1000/1000),
	)
}

// LogMetricForCount 日志调节
func LogMetricForCount(name string, count int) {
	var cname = C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.logMetricForCount(
		cname,
		C.int(count),
	)
}

// LogMetricForDuration 日志条件
func LogMetricForDuration(name string, durationInNano int64) {
	var cname = C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.logMetricForDuration(
		cname,
		C.ulonglong(durationInNano/1000/1000),
	)
}
