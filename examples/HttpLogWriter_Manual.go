package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	l4g "github.com/lerryxiao/log4go"
)

func dealGinLogger(c *gin.Context) {
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusOK, "OK")
		return
	}
	fmt.Printf("data: %s\n", string(data))
}

func getLogRecord() *l4g.LogRecord {
	message, _ := json.Marshal(map[string]string{
		"signature": "testSignature",
		"nonce":     "123456",
		"version":   "1.0",
	})

	return &l4g.LogRecord{
		Level:   l4g.FATAL,
		Source:  "source",
		Message: string(message),
		Created: time.Unix(time.Now().Unix(), 0).In(time.UTC),
	}
}

func getLogRecord2() *l4g.LogRecord {
	return &l4g.LogRecord{
		Level:   l4g.FATAL,
		Created: time.Unix(time.Now().Unix(), 0).In(time.UTC),
		Extend: []interface{}{
			l4g.EX_REPORT,
			"http://127.0.0.1:8080/logger",
			map[string]string{
				"appKey":    "IsD3UJ4Xgl",
				"from":      "sdk",
				"requestID": "1111111111111",
			},
			map[string]string{
				"signature": "testSignature",
				"nonce":     "654321",
				"version":   "2.0",
			},
		},
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	//router := gin.Default()
	router := gin.New()
	router.POST("/logger", dealGinLogger)

	go router.Run()

	httplog := l4g.NewHttpLogWriter("http://127.0.0.1:8080/logger", map[string]string{
		"appKey":    "IsD3UJ4Xgl",
		"from":      "sdk",
		"requestID": "xxxadfasefafa",
	}, 4)

	for i := 0; i < 2; i++ {
		httplog.LogWrite(getLogRecord())
	}

	for i := 0; i < 2; i++ {
		httplog.LogWrite(getLogRecord2())
	}

	httplog.Close()
}
