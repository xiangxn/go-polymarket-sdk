package polymarket

import (
	"context"
	"log"
	"testing"
)

func TestMarketMonitor(t *testing.T) {
	client := NewClient(DefaultConfig())
	marketMonitor := NewMarketMonitor("wss://ws-subscriptions-clob.polymarket.com", false, client, true)

	ch := marketMonitor.SubscribeResolved()
	ctx := context.Background()

	marketMonitor.SubscribeTokens([]string{}...)
	go marketMonitor.Run(ctx)

	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case info := <-ch:
				log.Printf("info: %+v", info)
			}

		}
	}()
}
