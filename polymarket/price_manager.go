package polymarket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

// ----------------------
// PriceManager 核心结构
// ----------------------

type PriceManager struct {
	mu                sync.RWMutex
	ws                *WebSocketClient
	callbacks         map[int]PriceUpdateCallback
	tokensPrice       map[string]*PriceData
	isConnecting      bool
	subsTokens        []string
	callbackIDCounter int
	clobMarketWSSURL  string
}

// NewPriceManager 创建新的 PriceManager 实例
func NewPriceManager(wsBaseUrl string) *PriceManager {
	return &PriceManager{
		callbacks:        make(map[int]PriceUpdateCallback),
		tokensPrice:      make(map[string]*PriceData),
		clobMarketWSSURL: fmt.Sprintf("%s/ws/market", wsBaseUrl),
	}
}

// Subscribe 订阅价格更新
func (pm *PriceManager) Subscribe(callback PriceUpdateCallback) func() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	callbackID := pm.callbackIDCounter
	pm.callbackIDCounter++
	pm.callbacks[callbackID] = callback

	return func() {
		pm.mu.Lock()
		defer pm.mu.Unlock()
		delete(pm.callbacks, callbackID)
	}
}

// Start 连接到 WebSocket, 开始处理数据
func (pm *PriceManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isConnecting || pm.ws != nil {
		return nil
	}

	pm.isConnecting = true

	log.Printf("🔌 连接到 WebSocket: %s", pm.clobMarketWSSURL)

	pm.ws = NewWebSocketClient(pm.clobMarketWSSURL, 10*time.Second)
	pm.ws.On("open", func(_ any) {
		log.Println("[PriceManager] ✅ WebSocket Connected")
		pm.isConnecting = false
		// 订阅市场数据
		pm.subscribeToMarket()
	})
	pm.ws.On("close", func(_ any) {
		log.Println("[PriceManager] 🔌 WebSocket Closed")
		pm.isConnecting = false
	})
	pm.ws.On("error", func(e any) {
		log.Println("[PriceManager] WebSocket Error:", e)
		pm.isConnecting = false
	})
	pm.ws.On("reconnect", func(_ any) {
		// 清空数据，防止旧数据异常
		pm.tokensPrice = make(map[string]*PriceData)
	})

	pm.ws.OnMessage(func(msg []byte) {
		// log.Println("[PriceManager] WebSocket Message:", string(msg))
		if string(msg) != "PONG" {
			pm.handleMessage(string(msg))
		}
	})

	pm.ws.Start()

	return nil
}

// SubscribeToMarket 订阅市场数据 (导出的方法)
func (pm *PriceManager) SubscribeToMarket(tokens ...string) {
	pm.subscribeToMarket(tokens...)
}

// subscribeToMarket 订阅市场数据
func (pm *PriceManager) subscribeToMarket(tokens ...string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(tokens) > 0 {
		pm.subsTokens = append(pm.subsTokens, tokens...)
		// 去重
		tokenSet := make(map[string]bool)
		for _, token := range pm.subsTokens {
			tokenSet[token] = true
		}
		pm.subsTokens = make([]string, 0, len(tokenSet))
		for token := range tokenSet {
			pm.subsTokens = append(pm.subsTokens, token)
		}
	}

	if len(pm.subsTokens) == 0 || pm.ws == nil {
		return
	}

	subscribeMessage := MarketMessage{
		Type:      "MARKET",
		AssetsIDs: pm.subsTokens,
	}

	data, _ := json.Marshal(subscribeMessage)
	err := pm.ws.Send(data)
	if err != nil {
		log.Printf("订阅市场失败: %v", err)
		return
	}

	log.Printf("📡 已订阅市场: %v", pm.subsTokens)
}

// UnsubscribeToMarket 取消订阅市场
func (pm *PriceManager) UnsubscribeToMarket(tokens ...string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(tokens) > 0 {
		for _, token := range tokens {
			// 从订阅列表中移除
			for i, t := range pm.subsTokens {
				if t == token {
					pm.subsTokens = append(pm.subsTokens[:i], pm.subsTokens[i+1:]...)
					break
				}
			}
			// 从价格数据中移除
			delete(pm.tokensPrice, token)
		}
	}
}

// handleMessage 处理 WebSocket 消息
func (pm *PriceManager) handleMessage(msg string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理消息时发生错误: %v", r)
		}
	}()

	eventType := gjson.Get(msg, "event_type").String()

	switch eventType {
	case "book":
		market := gjson.Get(msg, "market").String()
		Bids := gjson.Get(msg, "bids").Array()
		Asks := gjson.Get(msg, "asks").Array()
		assetID := gjson.Get(msg, "asset_id").String()
		timestamp := gjson.Get(msg, "timestamp").Int()

		if len(Bids) == 0 && len(Asks) == 0 {
			return
		}

		var bestBid orders.Book
		if len(Bids) > 0 {
			lastBid := Bids[len(Bids)-1]
			bestBid.Price = lastBid.Get("price").Float()
			bestBid.Size = lastBid.Get("size").Float()
		}

		var bestAsk orders.Book
		if len(Asks) > 0 {
			lastAsk := Asks[len(Asks)-1]
			bestAsk.Price = lastAsk.Get("price").Float()
			bestAsk.Size = lastAsk.Get("size").Float()
		}

		priceData := &PriceData{
			TokenID:   assetID,
			BestAsk:   &bestAsk,
			BestBid:   &bestBid,
			Market:    market,
			Timestamp: timestamp,
		}

		pm.updatePrice(priceData)
	case "last_trade_price":
		// 暂时什么也不做
	}
}

// updatePrice 更新价格并通知订阅者
func (pm *PriceManager) updatePrice(priceData *PriceData) {
	pm.mu.Lock()
	pm.tokensPrice[priceData.TokenID] = priceData
	currentPrice := priceData
	callbacks := make(map[int]PriceUpdateCallback)
	for id, callback := range pm.callbacks {
		callbacks[id] = callback
	}
	pm.mu.Unlock()

	// 通知所有订阅者
	for _, callback := range callbacks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("价格更新回调失败: %v", r)
				}
			}()
			callback(currentPrice)
		}()
	}
}

// GetCurrentPrice 获取当前价格
func (pm *PriceManager) GetCurrentPrice(tokenID string) *PriceData {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.tokensPrice[tokenID]
}

// Disconnect 断开连接
func (pm *PriceManager) Disconnect() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 关闭WebSocket连接
	if pm.ws != nil {
		pm.ws.Close()
		pm.ws = nil
	}

	// 清理数据
	pm.subsTokens = []string{}
	pm.callbacks = make(map[int]PriceUpdateCallback)
	pm.tokensPrice = make(map[string]*PriceData)
	pm.isConnecting = false
}
