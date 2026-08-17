package polymarket

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func slugFor(now time.Time) string {
	window := 5 * 60
	ts := now.Unix() / int64(window) * int64(window)
	return fmt.Sprintf("%s-%d", "btc-updown-5m", ts)
}

func TestMarketMonitor(t *testing.T) {
	client := NewClient(DefaultConfig())
	marketMonitor := NewMarketMonitor("wss://ws-subscriptions-clob.polymarket.com", false, client, true)

	slug := slugFor(time.Now())
	result, err := client.FetchMarketBySlug(slug)
	if err != nil {
		panic(err)
	}
	tokenIDs := utils.GetStringArray(result, "clobTokenIds")

	ch := marketMonitor.SubscribeResolved()
	ch2 := marketMonitor.SubscribeOrderBook()
	ctx := context.Background()

	marketMonitor.SubscribeTokens(tokenIDs...)
	go marketMonitor.Run(ctx)

	timer := time.NewTicker(20 * time.Second)

	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case info := <-ch:
				log.Printf("info: %+v", info)
			case book := <-ch2:
				log.Printf("book: %+v", book)
			case <-timer.C:
				marketMonitor.Reset(false)
				log.Println("======================================================================")
				time.Sleep(10 * time.Second)
				marketMonitor.SubscribeTokens(tokenIDs...)
			}

		}
	}()
}

// TestMarketMonitorHandleMessage_PriceChange 解析 price_change 消息并发送到通道(无需网络)
func TestMarketMonitorHandleMessage_PriceChange(t *testing.T) {
	mm := &MarketMonitor{
		orderBookCh:   make(chan *OrderBook, 8),
		priceChangeCh: make(chan *PriceChangeInfo, 8),
	}
	ch := mm.SubscribePriceChange()

	msg := []byte(`{
		"event_type": "price_change",
		"market": "0x5f65177b394277fd294cd75650044e32ba009a95022d88a0c1d565897d72f8f1",
		"price_changes": [
			{
				"asset_id": "71321045679252212594626385532706912750332728571942532289631379312455583992563",
				"price": "0.5",
				"size": "200",
				"side": "BUY",
				"hash": "56621a121a47ed9333273e21c83b660cff37ae50",
				"best_bid": "0.5",
				"best_ask": "1"
			}
		],
		"timestamp": "1757908892351"
	}`)
	mm.handleMessage(msg)

	select {
	case pc := <-ch:
		if pc.EventType != "price_change" {
			t.Fatalf("event_type mismatch: %s", pc.EventType)
		}
		if pc.Market != "0x5f65177b394277fd294cd75650044e32ba009a95022d88a0c1d565897d72f8f1" {
			t.Fatalf("market mismatch: %s", pc.Market)
		}
		if pc.Timestamp != 1757908892351 {
			t.Fatalf("timestamp mismatch: %d", pc.Timestamp)
		}
		if len(pc.PriceChanges) != 1 {
			t.Fatalf("price_changes len: %d", len(pc.PriceChanges))
		}
		item := pc.PriceChanges[0]
		if item.AssetID != "71321045679252212594626385532706912750332728571942532289631379312455583992563" ||
			item.Price != "0.5" || item.Size != "200" || item.Side != "BUY" ||
			item.Hash != "56621a121a47ed9333273e21c83b660cff37ae50" ||
			item.BestBid != "0.5" || item.BestAsk != "1" {
			t.Fatalf("price change item mismatch: %+v", item)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for price_change event")
	}
}

// TestMarketMonitorHandleMessage_NonPriceChangeIgnored book 等事件不应触发 price_change 通道(无需网络)
func TestMarketMonitorHandleMessage_NonPriceChangeIgnored(t *testing.T) {
	mm := &MarketMonitor{
		orderBookCh:   make(chan *OrderBook, 8),
		priceChangeCh: make(chan *PriceChangeInfo, 8),
	}

	ch := mm.SubscribePriceChange()

	mm.handleMessage([]byte(`{"event_type":"book","market":"mkt","asset_id":"a","bids":[],"asks":[]}`))

	select {
	case pc := <-ch:
		t.Fatalf("unexpected price_change: %+v", pc)
	default:
	}
}

// TestMarketMonitorHandleMessage_LastTradePrice 解析 last_trade_price 消息并发送到通道(无需网络)
func TestMarketMonitorHandleMessage_LastTradePrice(t *testing.T) {
	mm := &MarketMonitor{
		orderBookCh:      make(chan *OrderBook, 8),
		priceChangeCh:    make(chan *PriceChangeInfo, 8),
		lastTradePriceCh: make(chan *LastTradePriceInfo, 8),
	}

	msg := []byte(`{
		"event_type": "last_trade_price",
		"asset_id": "114122071509644379678018727908709560226618148003371446110114509806601493071694",
		"market": "0x6a67b9d828d53862160e470329ffea5246f338ecfffdf2cab45211ec578b0347",
		"price": "0.456",
		"size": "219.217767",
		"fee_rate_bps": "0",
		"side": "BUY",
		"timestamp": "1750428146322",
		"transaction_hash": "0xeeefffggghhh"
	}`)
	ch := mm.SubscribeLastTradePrice()

	mm.handleMessage(msg)

	select {
	case info := <-ch:
		if info.EventType != "last_trade_price" {
			t.Fatalf("event_type mismatch: %s", info.EventType)
		}
		if info.AssetID != "114122071509644379678018727908709560226618148003371446110114509806601493071694" {
			t.Fatalf("asset_id mismatch: %s", info.AssetID)
		}
		if info.Market != "0x6a67b9d828d53862160e470329ffea5246f338ecfffdf2cab45211ec578b0347" {
			t.Fatalf("market mismatch: %s", info.Market)
		}
		if info.Price != "0.456" || info.Size != "219.217767" || info.FeeRateBps != "0" ||
			info.Side != "BUY" || info.TransactionHash != "0xeeefffggghhh" || info.Timestamp != 1750428146322 {
			t.Fatalf("last_trade_price mismatch: %+v", info)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for last_trade_price event")
	}
}

// TestMarketMonitorHandleMessage_NoSubscriberSkip 未订阅的事件不应被解析/投递(无需网络)
func TestMarketMonitorHandleMessage_NoSubscriberSkip(t *testing.T) {
	mm := &MarketMonitor{
		orderBookCh:      make(chan *OrderBook, 8),
		resolvedCh:       make(chan *ResolvedInfo, 8),
		priceChangeCh:    make(chan *PriceChangeInfo, 8),
		lastTradePriceCh: make(chan *LastTradePriceInfo, 8),
	}

	mm.handleMessage([]byte(`{"event_type":"price_change","market":"mkt","price_changes":[{"asset_id":"a","price":"0.5"}],"timestamp":"1757908892351"}`))
	mm.handleMessage([]byte(`{"event_type":"last_trade_price","asset_id":"a","market":"mkt","price":"0.5","size":"1","side":"BUY","timestamp":"1757908892351"}`))
	mm.handleMessage([]byte(`{"event_type":"market_resolved","market":"mkt","timestamp":"1757908892351"}`))
	mm.handleMessage([]byte(`{"event_type":"book","market":"mkt","asset_id":"a","bids":[],"asks":[]}`))

	select {
	case pc := <-mm.priceChangeCh:
		t.Fatalf("unexpected price_change: %+v", pc)
	default:
	}
	select {
	case info := <-mm.lastTradePriceCh:
		t.Fatalf("unexpected last_trade_price: %+v", info)
	default:
	}
	select {
	case info := <-mm.resolvedCh:
		t.Fatalf("unexpected market_resolved: %+v", info)
	default:
	}
	select {
	case book := <-mm.orderBookCh:
		t.Fatalf("unexpected book: %+v", book)
	default:
	}

	if _, err := mm.GetTokenOrderBook("a"); err == nil {
		t.Fatal("book should not be stored without subscription or isStore")
	}
}

// TestMarketMonitorHandleMessage_BookStoredWithIsStore isStore 为 true 时,
// 即使没人订阅 orderBookCh,book 事件仍会被解析并存入快照(无需网络)
func TestMarketMonitorHandleMessage_BookStoredWithIsStore(t *testing.T) {
	mm := &MarketMonitor{
		orderBookCh: make(chan *OrderBook, 8),
		isStore:     true,
	}

	mm.handleMessage([]byte(`{
		"event_type":"book",
		"market":"mkt",
		"asset_id":"a",
		"bids":[{"price":"0.5","size":"10"}],
		"asks":[{"price":"0.6","size":"20"}],
		"timestamp":"1757908892351"
	}`))

	book, err := mm.GetTokenOrderBook("a")
	if err != nil {
		t.Fatal(err)
	}
	if book.Market != "mkt" || len(book.Bids) != 1 || len(book.Asks) != 1 {
		t.Fatalf("stored book mismatch: %+v", book)
	}
}

// TestMarketMonitorPriceChange 实盘订阅市场,验证能收到 price_change 事件
// 运行: https_proxy=http://127.0.0.1:1087 go test -v -run TestMarketMonitorPriceChange -timeout 120s ./polymarket/
func TestMarketMonitorPriceChange(t *testing.T) {
	client := NewClient(DefaultConfig())
	marketMonitor := NewMarketMonitor("wss://ws-subscriptions-clob.polymarket.com", false, client, true)

	slug := slugFor(time.Now())
	result, err := client.FetchMarketBySlug(slug)
	if err != nil {
		t.Fatal(err)
	}
	tokenIDs := utils.GetStringArray(result, "clobTokenIds")
	if len(tokenIDs) == 0 {
		t.Fatalf("no clobTokenIds for slug %s", slug)
	}

	ch := marketMonitor.SubscribePriceChange()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	marketMonitor.SubscribeTokens(tokenIDs...)
	go marketMonitor.Run(ctx)

	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()

	count := 0
	for {
		select {
		case <-timer.C:
			if count == 0 {
				t.Fatalf("no price_change received in 60s")
			}
			log.Printf("received %d price_change events", count)
			return
		case info := <-ch:
			count++
			if info.Market == "" || len(info.PriceChanges) == 0 {
				t.Fatalf("bad price_change: %+v", info)
			}
			if count <= 3 {
				log.Printf("price_change: market=%s changes=%d first=%+v",
					info.Market, len(info.PriceChanges), info.PriceChanges[0])
			}
		}
	}
}

// TestMarketMonitorLastTradePrice 实盘订阅市场,验证能收到 last_trade_price 事件
// 运行: https_proxy=http://127.0.0.1:1087 go test -v -run TestMarketMonitorLastTradePrice -timeout 120s ./polymarket/
func TestMarketMonitorLastTradePrice(t *testing.T) {
	client := NewClient(DefaultConfig())
	marketMonitor := NewMarketMonitor("wss://ws-subscriptions-clob.polymarket.com", false, client, true)

	slug := slugFor(time.Now())
	result, err := client.FetchMarketBySlug(slug)
	if err != nil {
		t.Fatal(err)
	}
	tokenIDs := utils.GetStringArray(result, "clobTokenIds")
	if len(tokenIDs) == 0 {
		t.Fatalf("no clobTokenIds for slug %s", slug)
	}

	ch := marketMonitor.SubscribeLastTradePrice()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	marketMonitor.SubscribeTokens(tokenIDs...)
	go marketMonitor.Run(ctx)

	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()

	count := 0
	for {
		select {
		case <-timer.C:
			if count == 0 {
				t.Fatalf("no last_trade_price received in 60s")
			}
			log.Printf("received %d last_trade_price events", count)
			return
		case info := <-ch:
			count++
			if info.Market == "" || info.AssetID == "" {
				t.Fatalf("bad last_trade_price: %+v", info)
			}
			if count <= 3 {
				log.Printf("last_trade_price: market=%s asset=%s price=%s size=%s side=%s",
					info.Market, info.AssetID, info.Price, info.Size, info.Side)
			}
		}
	}
}
