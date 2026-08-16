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
	case pc := <-mm.priceChangeCh:
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

	mm.handleMessage([]byte(`{"event_type":"book","market":"mkt","asset_id":"a","bids":[],"asks":[]}`))

	select {
	case pc := <-mm.priceChangeCh:
		t.Fatalf("unexpected price_change: %+v", pc)
	default:
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
