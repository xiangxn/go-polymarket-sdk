package polymarket

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type MarketMonitor struct {
	ws utils.WSClient
	// tokenId=>OrderBook
	orderBooks       map[string]*OrderBook
	mu               sync.RWMutex
	clobMarketWSSURL string
	subsTokens       []string
	muSubsTokens     sync.RWMutex
	pmClient         *PolymarketClient

	orderBookCh chan *OrderBook

	// 是否存储Orderbook,如果为false时，需要使用者在外部存储
	isStore bool
}

func NewMarketMonitor(wsBaseUrl string, isStore bool, client *PolymarketClient) *MarketMonitor {
	return &MarketMonitor{
		orderBooks:       make(map[string]*OrderBook),
		orderBookCh:      make(chan *OrderBook, 4096),
		clobMarketWSSURL: fmt.Sprintf("%s/ws/market", wsBaseUrl),
		pmClient:         client,
		isStore:          isStore,
	}
}

func (mm *MarketMonitor) Subscribe() <-chan *OrderBook {
	return mm.orderBookCh
}

func (mm *MarketMonitor) emitOrderBook(orderBook *OrderBook) {
	select {
	case mm.orderBookCh <- orderBook:
	default:
		log.Println("[MarketMonitor] fill channel full, dropping fill")
	}
}

func (pm *MarketMonitor) GetClient() *PolymarketClient {
	return pm.pmClient
}

// Run 启动 WebSocket 监听
func (pm *MarketMonitor) Run(ctx context.Context) error {
	log.Println("[MarketMonitor] Run start")
	defer log.Println("[MarketMonitor] Run exit")

	if pm.ws != nil && pm.ws.IsAlive() {
		return nil
	}

	pm.ws = utils.NewWSClient(utils.WSConfig{
		URL:          pm.clobMarketWSSURL,
		PingInterval: 10 * time.Second,
		Reconnect:    true,
		MaxReconnect: 20,
	}, pm)

	if err := pm.ws.Run(ctx); err != nil {
		pm.Disconnect()
		return err
	}

	pm.Disconnect()
	return ctx.Err()
}

// handleMessage 解析消息并发送 Tick
func (pm *MarketMonitor) handleMessage(msg string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MarketMonitor] handleMessage panic: %v", r)
		}
	}()

	eventType := gjson.Get(msg, "event_type").String()
	if eventType != "book" {
		return
	}

	var book OrderBook
	book.Market = gjson.Get(msg, "market").String()
	book.AssetId = gjson.Get(msg, "asset_id").String()
	book.Timestamp = gjson.Get(msg, "timestamp").Int()
	book.Latency = time.Now().UnixMilli() - book.Timestamp // 计算接收延迟
	bids := gjson.Get(msg, "bids").Array()
	for _, v := range bids {
		price, _ := strconv.ParseFloat(v.Get("price").String(), 64)
		size, _ := strconv.ParseFloat(v.Get("size").String(), 64)
		book.Bids = append(book.Bids, orders.Book{Price: price, Size: size})
	}
	asks := gjson.Get(msg, "asks").Array()
	for _, v := range asks {
		price, _ := strconv.ParseFloat(v.Get("price").String(), 64)
		size, _ := strconv.ParseFloat(v.Get("size").String(), 64)
		book.Asks = append(book.Asks, orders.Book{Price: price, Size: size})
	}

	// var bestBid, bestAsk orders.Book
	// if len(bids) > 0 {
	// 	// lastBid := Bids[len(Bids)-1]
	// 	lastBid := slices.MaxFunc(bids, func(a, b gjson.Result) int { return cmp.Compare(a.Get("price").Float(), b.Get("price").Float()) })
	// 	bestBid.Price = lastBid.Get("price").Float()
	// 	bestBid.Size = lastBid.Get("size").Float()
	// }
	// if len(asks) > 0 {
	// 	// lastAsk := Asks[len(Asks)-1]
	// 	lastAsk := slices.MinFunc(asks, func(a, b gjson.Result) int { return cmp.Compare(a.Get("price").Float(), b.Get("price").Float()) })
	// 	bestAsk.Price = lastAsk.Get("price").Float()
	// 	bestAsk.Size = lastAsk.Get("size").Float()
	// }
	pm.updateOrderBook(&book)
	pm.emitOrderBook(&book)
}

// Disconnect 断开 WS
func (pm *MarketMonitor) Disconnect() {
	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if pm.ws != nil {
		pm.ws.Close()
		pm.ws = nil
	}
	pm.subsTokens = nil
}

func (pm *MarketMonitor) Reset() {
	pm.muSubsTokens.Lock()
	pm.subsTokens = nil
	pm.muSubsTokens.Unlock()

	if pm.ws != nil {
		pm.ws.Reset()
	}
}

// SubscribeTokens 订阅市场数据 (导出的方法)
func (pm *MarketMonitor) SubscribeTokens(tokens ...string) {
	pm.subscribeToMarket(tokens...)
}

func (pm *MarketMonitor) UnsubscribeTokens(tokens ...string) {
	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if len(tokens) > 0 {
		for _, token := range tokens {
			// 从订阅列表中移除
			for i, t := range pm.subsTokens {
				if t == token {
					pm.subsTokens = append(pm.subsTokens[:i], pm.subsTokens[i+1:]...)
					break
				}
			}
		}
	}
}

func (pm *MarketMonitor) subscribeToMarket(tokens ...string) {
	pm.fetchOrderbooks(tokens...)

	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

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

	if len(pm.subsTokens) == 0 || pm.ws == nil || !pm.ws.IsAlive() {
		return
	}

	subscribeMessage := MarketMessage{
		Type:      "MARKET",
		AssetsIDs: pm.subsTokens,
	}

	data, _ := json.Marshal(subscribeMessage)
	err := pm.ws.Send(data)
	if err != nil {
		log.Printf("[MarketMonitor] 订阅市场失败: %v", err)
		return
	}

	log.Printf("[MarketMonitor] 📡 已订阅市场: %v", pm.subsTokens)
}

func (pm *MarketMonitor) updateOrderBook(orderBook *OrderBook) {
	if !pm.isStore {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	ob, ok := pm.orderBooks[orderBook.AssetId]
	if !ok {
		ob = &OrderBook{AssetId: orderBook.AssetId}
		pm.orderBooks[orderBook.AssetId] = ob

	}
	ob.Market = orderBook.Market
	ob.Timestamp = orderBook.Timestamp
	ob.Latency = orderBook.Latency
	ob.Bids = append(ob.Bids[:0], orderBook.Bids...)
	ob.Asks = append(ob.Asks[:0], orderBook.Asks...)
}

func (pm *MarketMonitor) fetchOrderbooks(tokens ...string) {
	if len(tokens) == 0 {
		return
	}
	var params []BookParams
	for _, token := range tokens {
		params = append(params, BookParams{TokenId: token})
	}
	orderBooks, err := pm.pmClient.GetOrderBooks(params)
	if err != nil {
		log.Printf("[MarketMonitor] 获取订单簿失败: %v", err)
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, orderBook := range orderBooks {
		ob := &OrderBook{
			Market:    orderBook.Market,
			AssetId:   orderBook.AssetId,
			Timestamp: orderBook.Timestamp,
			Latency:   time.Now().UnixMilli() - orderBook.Timestamp, // 计算接收延迟
		}
		ob.Bids = append(ob.Bids, orderBook.Bids...)
		ob.Asks = append(ob.Asks, orderBook.Asks...)
		pm.orderBooks[orderBook.AssetId] = ob
	}
}

func (pm *MarketMonitor) GetTokenOrderBook(tokenID string) (OrderBook, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if orderBook, ok := pm.orderBooks[tokenID]; ok {
		ob := *orderBook
		ob.Bids = append(ob.Bids, orderBook.Bids...)
		ob.Asks = append(ob.Asks, orderBook.Asks...)
		return ob, nil
	}
	return OrderBook{}, fmt.Errorf("[MarketMonitor] token price not found for %s", tokenID)
}

func (pm *MarketMonitor) GetTokenPrice(tokenID string) (*PriceData, error) {
	book, ok := pm.orderBooks[tokenID]
	if !ok {
		return nil, fmt.Errorf("[%s] is no data yet.", tokenID)
	}
	var bestBid, bestAsk orders.Book
	bids := book.Bids
	asks := book.Asks
	if len(bids) > 0 {
		// lastBid := Bids[len(Bids)-1]
		lastBid := slices.MaxFunc(bids, func(a, b orders.Book) int { return cmp.Compare(a.Price, b.Price) })
		bestBid.Price = lastBid.Price
		bestBid.Size = lastBid.Size
	}
	if len(asks) > 0 {
		// lastAsk := Asks[len(Asks)-1]
		lastAsk := slices.MinFunc(asks, func(a, b orders.Book) int { return cmp.Compare(a.Price, b.Price) })
		bestAsk.Price = lastAsk.Price
		bestAsk.Size = lastAsk.Size
	}
	var data PriceData
	data.TokenID = tokenID
	data.BestBid = &bestBid
	data.BestAsk = &bestAsk
	data.Market = book.Market
	data.Timestamp = book.Timestamp
	return &data, nil
}

/***WSClient handler实现***/

func (pm *MarketMonitor) OnOpen() {
	log.Println("[MarketMonitor] WebSocket Connected")
	pm.subscribeToMarket()
}

func (pm *MarketMonitor) OnReconnect() {
	log.Println("[MarketMonitor] WebSocket Reconnect...")
	// 清空数据，防止旧数据异常
	pm.mu.Lock()
	pm.orderBooks = make(map[string]*OrderBook)
	pm.mu.Unlock()

	pm.subscribeToMarket()
}

func (pm *MarketMonitor) OnError(err error) {
	log.Println("[MarketMonitor] WebSocket Error:", err)
}

func (pm *MarketMonitor) OnClose() {
	log.Println("[MarketMonitor] WebSocket Closed")
}

func (pm *MarketMonitor) OnMessage(msg []byte) {
	if string(msg) != "PONG" {
		pm.handleMessage(string(msg))
	}
}
