package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type BookStore struct {
	v          atomic.Value // *OrderBook
	lastUpdate int64
}

func (bs *BookStore) Load() *OrderBook {
	v := bs.v.Load()
	if v == nil {
		return nil
	}
	return v.(*OrderBook)
}

func (bs *BookStore) Store(book *OrderBook) {
	bs.lastUpdate = time.Now().UnixMilli()
	bs.v.Store(book)
}

type MarketMonitor struct {
	ws utils.WSClient

	// tokenId => immutable snapshot store
	orderBooks sync.Map

	clobMarketWSSURL     string
	customFeatureEnabled bool

	subsTokens   []string
	muSubsTokens sync.RWMutex

	pmClient *PolymarketClient

	// downstream consumer channel
	orderBookCh       chan *OrderBook
	resolvedCh        chan *ResolvedInfo
	priceChangeCh     chan *PriceChangeInfo
	lastTradePriceCh  chan *LastTradePriceInfo

	// 事件解析开关:仅在有人订阅后才解析对应事件,避免无人消费时的无效解析
	// book 事件额外受 isStore 控制,见 handleMessage
	orderBookSubscribed      atomic.Bool
	resolvedSubscribed       atomic.Bool
	priceChangeSubscribed    atomic.Bool
	lastTradePriceSubscribed atomic.Bool

	// 是否存储Orderbook
	isStore bool
}

func NewMarketMonitor(
	wsBaseUrl string,
	isStore bool,
	client *PolymarketClient,
	customFeatureEnabled bool,
) *MarketMonitor {

	return &MarketMonitor{
		orderBookCh:          make(chan *OrderBook, 4096),
		resolvedCh:           make(chan *ResolvedInfo, 4096),
		priceChangeCh:        make(chan *PriceChangeInfo, 4096),
		lastTradePriceCh:     make(chan *LastTradePriceInfo, 4096),
		clobMarketWSSURL:     fmt.Sprintf("%s/ws/market", wsBaseUrl),
		pmClient:             client,
		isStore:              isStore,
		customFeatureEnabled: customFeatureEnabled,
	}
}

func (mm *MarketMonitor) SubscribeOrderBook() <-chan *OrderBook {
	mm.orderBookSubscribed.Store(true)
	return mm.orderBookCh
}

func (mm *MarketMonitor) SubscribeResolved() <-chan *ResolvedInfo {
	mm.resolvedSubscribed.Store(true)
	return mm.resolvedCh
}

func (mm *MarketMonitor) SubscribePriceChange() <-chan *PriceChangeInfo {
	mm.priceChangeSubscribed.Store(true)
	return mm.priceChangeCh
}

func (mm *MarketMonitor) SubscribeLastTradePrice() <-chan *LastTradePriceInfo {
	mm.lastTradePriceSubscribed.Store(true)
	return mm.lastTradePriceCh
}

// 以下方法仅重置解析开关,channel 中已积压的事件不清理,由使用者自行处理。
// 重置后重新 Subscribe 即恢复解析。

// UnsubscribeOrderBook 停止解析 book 事件(注意:isStore 为 true 时仍会解析并入库)
func (mm *MarketMonitor) UnsubscribeOrderBook() {
	mm.orderBookSubscribed.Store(false)
}

// UnsubscribeResolved 停止解析 market_resolved 事件
func (mm *MarketMonitor) UnsubscribeResolved() {
	mm.resolvedSubscribed.Store(false)
}

// UnsubscribePriceChange 停止解析 price_change 事件
func (mm *MarketMonitor) UnsubscribePriceChange() {
	mm.priceChangeSubscribed.Store(false)
}

// UnsubscribeLastTradePrice 停止解析 last_trade_price 事件
func (mm *MarketMonitor) UnsubscribeLastTradePrice() {
	mm.lastTradePriceSubscribed.Store(false)
}

func (mm *MarketMonitor) emitOrderBook(book *OrderBook) {

	// drop oldest
	select {
	case mm.orderBookCh <- book:
		return

	default:
	}

	select {
	case <-mm.orderBookCh:
	default:
	}

	select {
	case mm.orderBookCh <- book:
	default:
		log.Println("[MarketMonitor] orderBookCh full")
	}
}

func (mm *MarketMonitor) emitPriceChange(info *PriceChangeInfo) {

	// drop oldest
	select {
	case mm.priceChangeCh <- info:
		return

	default:
	}

	select {
	case <-mm.priceChangeCh:
	default:
	}

	select {
	case mm.priceChangeCh <- info:
	default:
		log.Println("[MarketMonitor] priceChangeCh full")
	}
}

func (mm *MarketMonitor) emitLastTradePrice(info *LastTradePriceInfo) {

	// drop oldest
	select {
	case mm.lastTradePriceCh <- info:
		return

	default:
	}

	select {
	case <-mm.lastTradePriceCh:
	default:
	}

	select {
	case mm.lastTradePriceCh <- info:
	default:
		log.Println("[MarketMonitor] lastTradePriceCh full")
	}
}

func (pm *MarketMonitor) GetClient() *PolymarketClient {
	return pm.pmClient
}

// Run 启动 WS
func (pm *MarketMonitor) Run(ctx context.Context) error {

	log.Println("[MarketMonitor] Run start")
	defer log.Println("[MarketMonitor] Run exit")

	if pm.ws != nil && pm.ws.IsAlive() {
		return nil
	}

	pm.ws = utils.NewWSClient(
		utils.WSConfig{
			URL:            pm.clobMarketWSSURL,
			PingInterval:   10 * time.Second,
			Reconnect:      true,
			MaxReconnect:   20,
			MsgBufferSize:  32768,
			ReadBufferSize: 65536,
		},
		pm,
	)

	if err := pm.ws.Run(ctx); err != nil {
		pm.Disconnect()
		return err
	}

	pm.Disconnect()

	return ctx.Err()
}

// 高频热路径
func (pm *MarketMonitor) handleMessage(msg []byte) {

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MarketMonitor] handleMessage panic: %v", r)
		}
	}()

	result := gjson.ParseBytes(msg)

	event_type := result.Get("event_type").String()

	// 懒解析:对应事件无人订阅时直接跳过,省去 gjson 解析与结构体分配
	switch event_type {
	case "book":
		if pm.orderBookSubscribed.Load() || pm.isStore {
			pm.onOrderBook(&result)
		}
	case "market_resolved":
		if pm.resolvedSubscribed.Load() {
			pm.onMarketResolved(&result)
		}
	case "price_change":
		if pm.priceChangeSubscribed.Load() {
			pm.onPriceChange(&result)
		}
	case "last_trade_price":
		if pm.lastTradePriceSubscribed.Load() {
			pm.onLastTradePrice(&result)
		}
	default:
		// log.Printf("event_type: %s, %s", event_type, result.Get("winning_outcome").String())
	}
}

func (mm *MarketMonitor) onMarketResolved(info *gjson.Result) {
	resolvedInfo := &ResolvedInfo{
		EventType:      info.Get("event_type").String(),
		Id:             info.Get("id").String(),
		Market:         info.Get("market").String(),
		AssetsIds:      utils.GetStringArray(info, "assets_ids"),
		WinningAssetId: info.Get("winning_asset_id").String(),
		WinningOutcome: info.Get("winning_outcome").String(),
		Timestamp:      info.Get("timestamp").Int(),
		Tags:           utils.GetStringArray(info, "tags"),
	}

	select {
	case mm.resolvedCh <- resolvedInfo:
	default:
		log.Println("[MarketMonitor] resolvedCh full")
	}
}

func (pm *MarketMonitor) onOrderBook(info *gjson.Result) {
	book := &OrderBook{}
	book.Market = info.Get("market").String()
	book.AssetId = info.Get("asset_id").String()
	book.Timestamp = info.Get("timestamp").Int()
	book.Latency = time.Now().UnixMilli() - book.Timestamp

	// bids
	bids := info.Get("bids").Array()

	if len(bids) > 0 {
		book.Bids = make([]orders.Book, 0, len(bids))
		for _, v := range bids {
			book.Bids = append(book.Bids, orders.Book{
				Price: v.Get("price").Float(),
				Size:  v.Get("size").Float(),
			})
		}
	}

	// asks
	asks := info.Get("asks").Array()

	if len(asks) > 0 {
		book.Asks = make([]orders.Book, 0, len(asks))
		for _, v := range asks {
			book.Asks = append(book.Asks, orders.Book{
				Price: v.Get("price").Float(),
				Size:  v.Get("size").Float(),
			})
		}
	}

	pm.updateOrderBook(book)

	pm.emitOrderBook(book)
}

func (mm *MarketMonitor) onPriceChange(info *gjson.Result) {
	priceChange := &PriceChangeInfo{
		EventType: info.Get("event_type").String(),
		Market:    info.Get("market").String(),
		Timestamp: info.Get("timestamp").Int(),
	}

	changes := info.Get("price_changes").Array()

	if len(changes) > 0 {
		priceChange.PriceChanges = make([]PriceChangeItem, 0, len(changes))
		for _, v := range changes {
			priceChange.PriceChanges = append(priceChange.PriceChanges, PriceChangeItem{
				AssetID: v.Get("asset_id").String(),
				Price:   v.Get("price").String(),
				Size:    v.Get("size").String(),
				Side:    v.Get("side").String(),
				Hash:    v.Get("hash").String(),
				BestBid: v.Get("best_bid").String(),
				BestAsk: v.Get("best_ask").String(),
			})
		}
	}

	mm.emitPriceChange(priceChange)
}

func (mm *MarketMonitor) onLastTradePrice(info *gjson.Result) {
	lastTradePrice := &LastTradePriceInfo{
		EventType:       info.Get("event_type").String(),
		AssetID:         info.Get("asset_id").String(),
		Market:          info.Get("market").String(),
		Price:           info.Get("price").String(),
		Size:            info.Get("size").String(),
		FeeRateBps:      info.Get("fee_rate_bps").String(),
		Side:            info.Get("side").String(),
		TransactionHash: info.Get("transaction_hash").String(),
		Timestamp:       info.Get("timestamp").Int(),
	}

	mm.emitLastTradePrice(lastTradePrice)
}

// immutable snapshot store
func (pm *MarketMonitor) updateOrderBook(book *OrderBook) {

	if !pm.isStore {
		return
	}

	value, _ := pm.orderBooks.LoadOrStore(
		book.AssetId,
		&BookStore{},
	)

	store := value.(*BookStore)

	old := store.Load()

	// 防止多worker乱序回滚
	if old != nil {

		if book.Timestamp < old.Timestamp {
			return
		}
	}

	store.Store(book)
}

// Disconnect
func (pm *MarketMonitor) Disconnect() {

	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if pm.ws != nil {
		_ = pm.ws.Close()
		pm.ws = nil
	}

	pm.subsTokens = nil
}

// Reset
func (pm *MarketMonitor) Reset(isReconnect bool) {

	for _, t := range pm.subsTokens {
		pm.orderBooks.Delete(t)
	}

	pm.muSubsTokens.Lock()
	tokens := slices.Clone(pm.subsTokens)
	pm.subsTokens = nil
	pm.muSubsTokens.Unlock()

	pm.UnsubscribeTokens(tokens...)

	if pm.ws != nil && isReconnect {
		_ = pm.ws.Reset()
	}
}

func (pm *MarketMonitor) Subscribe() {
	if pm.ws == nil || !pm.ws.IsAlive() {
		return
	}

	// 先WS订阅
	subscribeMessage := MarketMessage{
		Type:                 "market",
		AssetsIDs:            []string{},
		CustomFeatureEnabled: pm.customFeatureEnabled,
	}

	data, _ := json.Marshal(subscribeMessage)
	if err := pm.ws.Send(data); err != nil {
		log.Printf("[MarketMonitor] subscribe failed: %v\n%s", err, data)
		return
	}
}

// SubscribeTokens
func (pm *MarketMonitor) SubscribeTokens(tokens ...string) {
	pm.subscribeMarket(tokens...)
}

func (pm *MarketMonitor) UnsubscribeTokens(tokens ...string) {

	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if len(tokens) == 0 {
		return
	}

	tokenSet := make(map[string]struct{}, len(tokens))

	if pm.isStore {
		for _, t := range tokens {
			tokenSet[t] = struct{}{}
			pm.orderBooks.Delete(t)
		}
	} else {
		for _, t := range tokens {
			tokenSet[t] = struct{}{}
		}
	}

	dst := pm.subsTokens[:0]

	for _, t := range pm.subsTokens {

		if _, remove := tokenSet[t]; !remove {
			dst = append(dst, t)
		}
	}

	pm.subsTokens = dst

	if len(dst) <= 0 || pm.ws == nil || !pm.ws.IsAlive() {
		return
	}

	subscribeMessage := DynamicSubMarketMessage{
		Operation:            DynamicUnSub,
		AssetsIds:            tokens,
		CustomFeatureEnabled: &pm.customFeatureEnabled,
	}

	data, _ := json.Marshal(subscribeMessage)
	if err := pm.ws.Send(data); err != nil {
		log.Printf("[MarketMonitor] subscribe failed: %v\n%s", err, data)
		return
	}

	log.Printf("[MarketMonitor] unsubscribed market tokens: %v", tokens)
}

func (pm *MarketMonitor) subscribeMarket(tokens ...string) {

	pm.muSubsTokens.Lock()

	if len(tokens) > 0 {

		pm.subsTokens = append(pm.subsTokens, tokens...)

		// dedup
		set := make(map[string]struct{})

		dst := pm.subsTokens[:0]

		for _, t := range pm.subsTokens {

			if _, ok := set[t]; ok {
				continue
			}

			set[t] = struct{}{}

			dst = append(dst, t)
		}

		pm.subsTokens = dst
	}

	subs := append([]string(nil), pm.subsTokens...)

	pm.muSubsTokens.Unlock()

	if len(subs) <= 0 || pm.ws == nil || !pm.ws.IsAlive() {
		return
	}

	// 动态订阅tokens
	subscribeMessage := DynamicSubMarketMessage{
		Operation:            DynamicSub,
		AssetsIds:            subs,
		CustomFeatureEnabled: &pm.customFeatureEnabled,
	}

	data, _ := json.Marshal(subscribeMessage)
	if err := pm.ws.Send(data); err != nil {
		log.Printf("[MarketMonitor] subscribe failed: %v\n%s", err, data)
		return
	}

	log.Printf("[MarketMonitor] subscribed market tokens: %v", subs)

	// 异步REST补快照，暂时无意义，所以注释掉
	//go pm.fetchOrderbooks(subs...)
}

// RemoveTokenOrderBook 按 tokenId 清理 orderBooks 中的条目
func (pm *MarketMonitor) RemoveTokenOrderBook(tokenID string) {
	pm.orderBooks.Delete(tokenID)
}

// immutable pointer
func (pm *MarketMonitor) GetTokenOrderBook(tokenID string) (*OrderBook, error) {

	value, ok := pm.orderBooks.Load(tokenID)

	if !ok {
		return nil, fmt.Errorf("[MarketMonitor] token not found: %s", tokenID)
	}

	store := value.(*BookStore)
	book := store.Load()

	if book == nil {
		return nil, fmt.Errorf("[MarketMonitor] token empty: %s", tokenID)
	}

	return book, nil
}

// 高频读取路径
func (pm *MarketMonitor) GetTokenPrice(tokenID string) (*PriceData, error) {

	book, err := pm.GetTokenOrderBook(tokenID)

	if err != nil {
		return nil, err
	}

	var bestBid *orders.Book
	var bestAsk *orders.Book

	if len(book.Bids) > 0 {
		bestBid = &book.Bids[len(book.Bids)-1]
	}

	if len(book.Asks) > 0 {
		bestAsk = &book.Asks[len(book.Asks)-1]
	}

	return &PriceData{
		TokenID:   tokenID,
		BestBid:   bestBid,
		BestAsk:   bestAsk,
		Market:    book.Market,
		Timestamp: book.Timestamp,
	}, nil
}

/*** WSClient handlers ***/

func (pm *MarketMonitor) OnOpen() {
	log.Println("[MarketMonitor] WebSocket connected")

	// pm.Subscribe()

	pm.subscribeMarket()
}

func (pm *MarketMonitor) OnReconnect() {
	log.Println("[MarketMonitor] WebSocket reconnect")
	pm.subscribeMarket()
}

func (pm *MarketMonitor) OnError(err error) {
	log.Println("[MarketMonitor] WebSocket error:", err)
}

func (pm *MarketMonitor) OnClose() {
	log.Println("[MarketMonitor] WebSocket closed")
}

func (pm *MarketMonitor) OnMessage(msg []byte) {
	// 高频零alloc heartbeat
	if len(msg) == 4 &&
		msg[0] == 'P' &&
		msg[1] == 'O' &&
		msg[2] == 'N' &&
		msg[3] == 'G' {
		return
	}

	// 直接在本 goroutine 处理消息。
	// OnMessage 由 WSClient 的 messageLoop worker pool 并发调用，
	// 不需要再经过额外的 channel / worker pool 中转，避免了无意义
	// 的 make+copy 内存分配和二次 channel 调度，从根本解决 slow consumer 问题。
	pm.handleMessage(msg)
}
