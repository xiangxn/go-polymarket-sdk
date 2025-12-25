package tests

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func TestPriceManager(t *testing.T) {
	config := polymarket.DefaultConfig()
	privateKey := os.Getenv("SIGNERKEY")
	polymarketClient := polymarket.NewClient(privateKey, config)
	for {
		marketSlug := fmt.Sprintf("eth-updown-15m-%d", utils.RoundTo15Minutes())
		log.Printf("%s/markets/slug/%s", config.Polymarket.GammaBaseURL, marketSlug)

		market, err := polymarketClient.FetchMarketBySlug(marketSlug)
		if err != nil {
			log.Fatal("FetchMarketBySlug failed:", err)
			return
		}

		endData, err := utils.ToTimestamp(market.Get("endDate").String())
		if err != nil {
			log.Fatal("ToTimestamp failed:", err)
			return
		}

		var tokenIds []string
		gjson.Parse(market.Get("clobTokenIds").String()).ForEach(func(key, value gjson.Result) bool {
			tokenIds = append(tokenIds, value.String())
			return true
		})

		// 启动价格监听
		pm := polymarket.NewPriceManager(config.Polymarket.ClobWSBaseSURL)
		pm.SubscribeToMarket(tokenIds...)
		pm.Subscribe(func(priceData *polymarket.PriceData) {
			// log.Printf("book: %+v", priceData)
			now := time.Now().UnixMilli()
			token0 := pm.GetCurrentPrice(tokenIds[0])
			token1 := pm.GetCurrentPrice(tokenIds[1])
			if token0 == nil || token1 == nil || token0.BestAsk.Price == 0 || token1.BestAsk.Price == 0 {
				return
			}
			if token0.BestAsk.Price+token1.BestAsk.Price < 1.0 {
				log.Printf("Book Data === BestAsk: %.2f/%.2f=%.2f, %.2f/%.2f, delay: %d", token0.BestAsk.Price, token1.BestAsk.Price, token0.BestAsk.Price+token1.BestAsk.Price, token0.BestAsk.Size, token1.BestAsk.Size, now-priceData.Timestamp)
			}
			if now-endData > 1000 {
				// 停止监听
				pm.Disconnect()
			}
		})
		if err := pm.Start(); err != nil {
			log.Fatal("PriceManager start failed:", err)
		}

		for time.Now().UnixMilli() <= endData+1000 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
