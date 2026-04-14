package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	pmModel "github.com/polymarket/go-order-utils/pkg/model"
	"github.com/tidwall/gjson"
	sdkModel "github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

// Fill 表示一笔成交明细
type Fill struct {
	FillID   string // 平台返回的 trade id
	OrderID  string
	MarketID string
	TokenID  string
	Status   string

	Side  pmModel.Side
	Price float64
	Size  float64

	Fee  float64
	Time int64
}

// TradeMonitor 监听用户成交事件
type TradeMonitor struct {
	ws             utils.WSClient
	creds          *sdkModel.ApiKeyCreds
	clobUserWSSURL string

	fillCh chan Fill
}

func NewTradeMonitor(wsBaseURL string, creds *sdkModel.ApiKeyCreds) *TradeMonitor {
	return &TradeMonitor{
		creds:          creds,
		clobUserWSSURL: fmt.Sprintf("%s/ws/user", wsBaseURL),
		fillCh:         make(chan Fill, 4096),
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

func (tm *TradeMonitor) Subscribe() <-chan Fill {
	return tm.fillCh
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
	tm.subscribeUserTrade()
}

func (tm *TradeMonitor) OnError(err error) {
	log.Println("[TradeMonitor] WebSocket Error:", err)
}

func (tm *TradeMonitor) OnClose() {
	log.Println("[TradeMonitor] WebSocket Closed")
}

func (tm *TradeMonitor) OnMessage(msg []byte) {
	if string(msg) != "PONG" {
		tm.handleMessage(msg)
	}
}

func (tm *TradeMonitor) subscribeUserTrade() {
	if tm.ws == nil || !tm.ws.IsAlive() {
		return
	}

	subscribeMessage := sdkModel.SubscribeUserMessage{
		Type:    "user",
		Markets: []string{},
		Auth:    tm.getAuth(),
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

func (tm *TradeMonitor) isTargetOwner(owner string) bool {
	return strings.EqualFold(owner, tm.creds.Key)
}

func (tm *TradeMonitor) emitFill(fill Fill) {
	select {
	case tm.fillCh <- fill:
	default:
		log.Println("[TradeMonitor] fill channel full, dropping fill")
	}
}

func (tm *TradeMonitor) procTrade(msg []byte) {
	// MATCHED, MINED, CONFIRMED, RETRYING, FAILED
	var wsTrade sdkModel.WSTrade
	if err := json.Unmarshal(msg, &wsTrade); err != nil {
		log.Printf("[TradeMonitor] handleMessage json.Unmarshal error: %v", err)
		return
	}
	// log.Printf("wsTrade: %+v", wsTrade)

	if wsTrade.Status != "MINED" && wsTrade.Status != "FAILED" {
		return
	}

	baseFill := Fill{
		FillID:   wsTrade.Id,
		MarketID: wsTrade.Market,
		Status:   wsTrade.Status,
		Time:     wsTrade.Timestamp,
	}

	// taker
	if tm.isTargetOwner(wsTrade.Owner) {
		side := pmModel.BUY
		if wsTrade.Side == string(orders.SELL) {
			side = pmModel.SELL
		}

		fill := baseFill
		fill.OrderID = wsTrade.TakerOrderId
		fill.TokenID = wsTrade.AssetId
		fill.Side = side
		fill.Price = wsTrade.Price
		fill.Size = wsTrade.Size
		fill.Fee = wsTrade.FeeRateBps * wsTrade.Size * wsTrade.Price
		tm.emitFill(fill)
	}

	// maker
	for _, mo := range wsTrade.MakerOrders {
		if !tm.isTargetOwner(mo.Owner) {
			continue
		}
		side := pmModel.BUY
		if mo.Side == string(orders.SELL) {
			side = pmModel.SELL
		}

		fill := baseFill
		fill.OrderID = mo.OrderId
		fill.TokenID = mo.AssetId
		fill.Side = side
		fill.Price = mo.Price
		fill.Size = mo.MatchedAmount
		fill.Fee = mo.FeeRateBps * mo.MatchedAmount * mo.Price
		tm.emitFill(fill)
	}
}

func (tm *TradeMonitor) handleMessage(msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TradeMonitor] handleMessage panic: %v", r)
		}
	}()

	eventType := gjson.GetBytes(msg, "event_type").String()
	if eventType == "trade" {
		tm.procTrade(msg)
	} else if eventType == "order" {
		tm.procOrder(msg)
	}

}

func (tm *TradeMonitor) procOrder(msg []byte) {
	var wsOrder sdkModel.WSOrder
	if err := json.Unmarshal(msg, &wsOrder); err != nil {
		log.Printf("[TradeMonitor] handleMessage json.Unmarshal error: %+v", err)
		return
	}

	if tm.isTargetOwner(wsOrder.Owner) && wsOrder.Type == "CANCELLATION" {
		baseFill := Fill{
			FillID:   wsOrder.Id,
			MarketID: wsOrder.Market,
			Status:   wsOrder.Status, //'LIVE', 'MATCHED', 'CANCELED'
			Time:     wsOrder.Timestamp,
		}
		side := pmModel.BUY
		if wsOrder.Side == string(orders.SELL) {
			side = pmModel.SELL
		}
		fill := baseFill
		fill.OrderID = wsOrder.Id
		fill.TokenID = wsOrder.AssetId
		fill.Side = side
		fill.Price = wsOrder.Price
		fill.Size = wsOrder.OriginalSize
		fill.Fee = 0
		tm.emitFill(fill)
	}
}
