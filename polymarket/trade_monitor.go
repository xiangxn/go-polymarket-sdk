package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	sdkModel "github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type TradeEventType string

const (
	TradeEventTypeTrade   TradeEventType = "trade"
	TradeEventTypeOrder   TradeEventType = "order"
	TradeEventTypeUnknown TradeEventType = "unknown"
)

type TradeEvent struct {
	EventType  TradeEventType
	ReceivedAt int64
	Raw        json.RawMessage

	Trade *sdkModel.WSTrade
	Order *sdkModel.WSOrder

	ParseErr error
}

// TradeMonitor 监听用户交易相关事件
// 对齐官方 user channel：全量透传 trade/order 事件，不做状态过滤。
type TradeMonitor struct {
	ws             utils.WSClient
	creds          *sdkModel.ApiKeyCreds
	clobUserWSSURL string

	eventCh chan TradeEvent
}

func NewTradeMonitor(wsBaseURL string, creds *sdkModel.ApiKeyCreds) *TradeMonitor {
	return &TradeMonitor{
		creds:          creds,
		clobUserWSSURL: fmt.Sprintf("%s/ws/user", wsBaseURL),
		eventCh:        make(chan TradeEvent, 4096),
	}
}

func (tm *TradeMonitor) Run(ctx context.Context) error {
	log.Println("[TradeMonitor] Run start")
	defer log.Println("[TradeMonitor] Run exit")

	if tm.ws != nil && tm.ws.IsAlive() {
		return nil
	}

	tm.ws = utils.NewWSClient(utils.WSConfig{
		URL:          tm.clobUserWSSURL,
		PingInterval: 10 * time.Second,
		Reconnect:    true,
		MaxReconnect: 20,
	}, tm)

	return tm.ws.Run(ctx)
}

func (tm *TradeMonitor) SubscribeEvents() <-chan TradeEvent {
	return tm.eventCh
}

func (tm *TradeMonitor) Close() error {
	if tm.ws == nil {
		return nil
	}
	return tm.ws.Close()
}

func (tm *TradeMonitor) OnOpen() {
	log.Println("[TradeMonitor] WebSocket Connected")
	tm.subscribeUserTrade()
}

func (tm *TradeMonitor) OnReconnect() {
	log.Println("[TradeMonitor] WebSocket Reconnect...")
	tm.subscribeUserTrade()
}

func (tm *TradeMonitor) OnError(err error) {
	log.Println("[TradeMonitor] WebSocket Error:", err)
}

func (tm *TradeMonitor) OnClose() {
	log.Println("[TradeMonitor] WebSocket Closed")
}

func (tm *TradeMonitor) OnMessage(msg []byte) {
	if isHeartbeatMessage(msg) {
		return
	}
	tm.handleMessage(msg)
}

func (tm *TradeMonitor) subscribeUserTrade(markets ...string) {
	if tm.ws == nil || !tm.ws.IsAlive() {
		return
	}

	subscribeMessage := map[string]any{
		"type": "user",
		"auth": tm.getAuth(),
	}
	if len(markets) > 0 {
		subscribeMessage["markets"] = markets
	}

	data, _ := json.Marshal(subscribeMessage)
	if err := tm.ws.Send(data); err != nil {
		log.Printf("[TradeMonitor] 订阅User交易失败: %v", err)
	}
}

func (tm *TradeMonitor) getAuth() *sdkModel.WSUserAuth {
	if tm.creds == nil {
		return nil
	}
	return &sdkModel.WSUserAuth{
		APIKey:     tm.creds.Key,
		Secret:     tm.creds.Secret,
		Passphrase: tm.creds.Passphrase,
	}
}

func (tm *TradeMonitor) emitEvent(ev TradeEvent) {
	select {
	case tm.eventCh <- ev:
	default:
		log.Println("[TradeMonitor] event channel full, dropping event")
	}
}

func (tm *TradeMonitor) procTrade(msg []byte) {
	var wsTrade sdkModel.WSTrade
	if err := json.Unmarshal(msg, &wsTrade); err != nil {
		tm.emitEvent(TradeEvent{
			EventType:  TradeEventTypeTrade,
			ReceivedAt: time.Now().UnixMilli(),
			Raw:        append([]byte(nil), msg...),
			ParseErr:   err,
		})
		return
	}

	tm.emitEvent(TradeEvent{
		EventType:  TradeEventTypeTrade,
		ReceivedAt: time.Now().UnixMilli(),
		Raw:        append([]byte(nil), msg...),
		Trade:      &wsTrade,
	})
}

func (tm *TradeMonitor) handleMessage(msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TradeMonitor] handleMessage panic: %v", r)
		}
	}()

	eventType := gjson.GetBytes(msg, "event_type").String()
	switch eventType {
	case string(TradeEventTypeTrade):
		tm.procTrade(msg)
	case string(TradeEventTypeOrder):
		tm.procOrder(msg)
	default:
		tm.emitEvent(TradeEvent{
			EventType:  TradeEventTypeUnknown,
			ReceivedAt: time.Now().UnixMilli(),
			Raw:        append([]byte(nil), msg...),
		})
	}
}

func (tm *TradeMonitor) procOrder(msg []byte) {
	var wsOrder sdkModel.WSOrder
	if err := json.Unmarshal(msg, &wsOrder); err != nil {
		tm.emitEvent(TradeEvent{
			EventType:  TradeEventTypeOrder,
			ReceivedAt: time.Now().UnixMilli(),
			Raw:        append([]byte(nil), msg...),
			ParseErr:   err,
		})
		return
	}

	tm.emitEvent(TradeEvent{
		EventType:  TradeEventTypeOrder,
		ReceivedAt: time.Now().UnixMilli(),
		Raw:        append([]byte(nil), msg...),
		Order:      &wsOrder,
	})
}

func isHeartbeatMessage(msg []byte) bool {
	trimmed := strings.TrimSpace(string(msg))
	return trimmed == "" || trimmed == "PONG" || trimmed == "{}"
}
