package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	orderBooks map[string]*BookStore
	mu         sync.RWMutex

	clobMarketWSSURL string

	subsTokens   []string
	muSubsTokens sync.RWMutex

	pmClient *PolymarketClient

	// downstream consumer channel
	orderBookCh chan *OrderBook

	// 是否存储Orderbook
	isStore bool
}

func NewMarketMonitor(
	wsBaseUrl string,
	isStore bool,
	client *PolymarketClient,
) *MarketMonitor {

	return &MarketMonitor{
		orderBooks:       make(map[string]*BookStore),
		orderBookCh:      make(chan *OrderBook, 4096),
		clobMarketWSSURL: fmt.Sprintf("%s/ws/market", wsBaseUrl),
		pmClient:         client,
		isStore:          isStore,
	}
}

func (mm *MarketMonitor) Subscribe() <-chan *OrderBook {
	return mm.orderBookCh
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
			URL:          pm.clobMarketWSSURL,
			PingInterval: 10 * time.Second,
			Reconnect:    true,
			MaxReconnect: 20,
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

	if result.Get("event_type").String() != "book" {
		return
	}

	book := &OrderBook{}
	book.Market = result.Get("market").String()
	book.AssetId = result.Get("asset_id").String()
	book.Timestamp = result.Get("timestamp").Int()
	book.Latency = time.Now().UnixMilli() - book.Timestamp

	// bids
	bids := result.Get("bids").Array()

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
	asks := result.Get("asks").Array()

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

// immutable snapshot store
func (pm *MarketMonitor) updateOrderBook(book *OrderBook) {

	if !pm.isStore {
		return
	}

	pm.mu.RLock()

	store, ok := pm.orderBooks[book.AssetId]

	pm.mu.RUnlock()

	if !ok {

		pm.mu.Lock()
		store, ok = pm.orderBooks[book.AssetId]
		if !ok {
			store = &BookStore{}
			pm.orderBooks[book.AssetId] = store
		}
		pm.mu.Unlock()
	}

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
func (pm *MarketMonitor) Reset() {

	pm.mu.Lock()
	for _, t := range pm.subsTokens {
		delete(pm.orderBooks, t)
	}
	pm.mu.Unlock()

	pm.muSubsTokens.Lock()
	pm.subsTokens = nil
	pm.muSubsTokens.Unlock()

	if pm.ws != nil {
		_ = pm.ws.Reset()
	}
}

// SubscribeTokens
func (pm *MarketMonitor) SubscribeTokens(tokens ...string) {
	pm.subscribeToMarket(tokens...)
}

func (pm *MarketMonitor) UnsubscribeTokens(tokens ...string) {

	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if len(tokens) == 0 {
		return
	}

	tokenSet := make(map[string]struct{}, len(tokens))

	if pm.isStore {
		pm.mu.Lock()
		for _, t := range tokens {
			tokenSet[t] = struct{}{}
			delete(pm.orderBooks, t)
		}
		pm.mu.Unlock()
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
}

func (pm *MarketMonitor) subscribeToMarket(tokens ...string) {

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

	if len(subs) == 0 ||
		pm.ws == nil ||
		!pm.ws.IsAlive() {
		return
	}

	// 先WS订阅
	subscribeMessage := MarketMessage{
		Type:      "MARKET",
		AssetsIDs: subs,
	}

	data, _ := json.Marshal(subscribeMessage)

	if err := pm.ws.Send(data); err != nil {
		log.Printf("[MarketMonitor] subscribe failed: %v", err)
		return
	}

	log.Printf("[MarketMonitor] subscribed markets: %v", subs)

	// 异步REST补快照，暂时无意义，所以注释掉
	//go pm.fetchOrderbooks(subs...)
}

// REST snapshot
func (pm *MarketMonitor) fetchOrderbooks(tokens ...string) {

	if len(tokens) == 0 {
		return
	}

	params := make([]BookParams, 0, len(tokens))

	for _, token := range tokens {

		params = append(params, BookParams{
			TokenId: token,
		})
	}

	orderBooks, err := pm.pmClient.GetOrderBooks(params)

	if err != nil {

		log.Printf(
			"[MarketMonitor] fetch orderbooks failed: %v",
			err,
		)

		return
	}

	for _, src := range orderBooks {

		book := &OrderBook{
			Market:    src.Market,
			AssetId:   src.AssetId,
			Timestamp: src.Timestamp,
			Latency:   time.Now().UnixMilli() - src.Timestamp,
		}

		if len(src.Bids) > 0 {

			book.Bids =
				make([]orders.Book, len(src.Bids))

			copy(book.Bids, src.Bids)
		}

		if len(src.Asks) > 0 {

			book.Asks =
				make([]orders.Book, len(src.Asks))

			copy(book.Asks, src.Asks)
		}

		pm.updateOrderBook(book)
	}
}

// immutable pointer
func (pm *MarketMonitor) GetTokenOrderBook(tokenID string) (*OrderBook, error) {

	pm.mu.RLock()

	store, ok := pm.orderBooks[tokenID]

	pm.mu.RUnlock()

	if !ok {
		return nil,
			fmt.Errorf(
				"[MarketMonitor] token not found: %s",
				tokenID,
			)
	}

	book := store.Load()

	if book == nil {
		return nil,
			fmt.Errorf(
				"[MarketMonitor] token empty: %s",
				tokenID,
			)
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
	log.Println("[MarketMonitor] WS connected")
	pm.subscribeToMarket()
}

func (pm *MarketMonitor) OnReconnect() {
	log.Println("[MarketMonitor] WS reconnect")
	pm.subscribeToMarket()
}

func (pm *MarketMonitor) OnError(err error) {
	log.Println("[MarketMonitor] WS error:", err)
}

func (pm *MarketMonitor) OnClose() {
	log.Println("[MarketMonitor] WS closed")
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

	pm.handleMessage(msg)
}
