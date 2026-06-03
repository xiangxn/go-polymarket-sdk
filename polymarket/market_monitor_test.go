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
