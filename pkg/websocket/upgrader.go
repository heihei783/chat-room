package websocket

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境应该验证Origin
		return true
	},
	HandshakeTimeout: 10 * time.Second,
}

// Configure 根据配置设置WebSocket连接参数（由main在启动时调用）
// 同时将ping/pong周期与消息大小上限同步到读写泵，使config.yaml中的websocket段真正生效
func Configure(readBuffer, writeBuffer int, handshakeTimeout, pingPeriodCfg, pongWaitCfg time.Duration, maxMessageSizeCfg int64, checkOrigin func(*http.Request) bool) {
	Upgrader.ReadBufferSize = readBuffer
	Upgrader.WriteBufferSize = writeBuffer
	Upgrader.HandshakeTimeout = handshakeTimeout
	if checkOrigin != nil {
		Upgrader.CheckOrigin = checkOrigin
	}

	// 仅当配置值合法（>0）时覆盖默认值
	if pingPeriodCfg > 0 {
		pingPeriod = pingPeriodCfg
	}
	if pongWaitCfg > 0 {
		pongWait = pongWaitCfg
	}
	if maxMessageSizeCfg > 0 {
		maxMessageSize = maxMessageSizeCfg
	}
}
