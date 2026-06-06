package utils

import (
	"context"
	"log"
	"testing"
	"time"
)

type TestPingPong struct {
	ws WSClient
}

func (tp *TestPingPong) OnMessage(msg []byte) {
	log.Printf("msg: %s", msg)
}

func (tp *TestPingPong) OnOpen() {
	log.Println("[MarketMonitor] WebSocket connected")

}

func (tp *TestPingPong) OnReconnect() {
	log.Println("[MarketMonitor] WebSocket reconnect")
}

func (tp *TestPingPong) OnError(err error) {
	log.Println("[MarketMonitor] WebSocket error:", err)
}

func (tp *TestPingPong) OnClose() {
	log.Println("[MarketMonitor] WebSocket closed")
}

func (tp *TestPingPong) Run(ctx context.Context) error {
	tp.ws = NewWSClient(
		WSConfig{
			URL:          "wss://ws-subscriptions-clob.polymarket.com/ws/market",
			PingInterval: 10 * time.Second,
			Reconnect:    true,
			MaxReconnect: 20,
			// TextHeartbeat:    true,
			// TextHeartbeatMsg: []byte("PING"),
		},
		tp,
	)
	if err := tp.ws.Run(ctx); err != nil {
		return err
	}

	return ctx.Err()
}
func TestPing(t *testing.T) {
	tpp := TestPingPong{}
	err := tpp.Run(t.Context())
	if err != nil {
		log.Printf("err: %v", err)
	}
}
