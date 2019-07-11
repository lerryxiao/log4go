package gcat

import (
	"github.com/jslyzt/gocat/ccat"
)

// Config 配置
type Config struct {
	EncoderType     int
	EnableHeartbeat int
	EnableSampling  int
	EnableDebugLog  int
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{ ENCoderBinary, 1, 1, 0, }
}

// DefaultConfigForCat2 cat2默认配置
func DefaultConfigForCat2() Config {
	return Config{ ENCoderText, 1, 0, 0, }
}

// Init 初始化
func Init(domain string, configs ...Config) {
	var config Config;
	if len(configs) > 1 {
		panic("Only 1 config can be specified while initializing cat.")
	} else if len(configs) == 1 {
		config = configs[0]
	} else {
		config = DefaultConfig()
	}

	ccat.InitWithConfig(domain, ccat.BuildConfig(
		config.EncoderType,
		config.EnableHeartbeat,
		config.EnableSampling,
		config.EnableDebugLog,
	))

	go ccat.Background()
}

// Shutdown 关闭
func Shutdown() {
	ccat.Shutdown()
}

// Wait 等待
func Wait() {
	ccat.Wait()
}
