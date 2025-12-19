package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

// RoundTo15Minutes 将时间向下舍入到最近的15分钟边界，返回Unix时间戳（秒）
func RoundTo15Minutes(date ...time.Time) int64 {
	var d time.Time
	if len(date) == 0 {
		d = time.Now()
	} else {
		d = date[0]
	}

	minutes := d.Minute()
	floored := (minutes / 15) * 15

	rounded := time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), floored, 0, 0, d.Location())
	return rounded.Unix()
}

func ToTimestamp(dateStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func main() {
	polymarketClient := polymarket.NewClient("", polymarket.Config{})
	for {
		marketSlug := fmt.Sprintf("eth-updown-15m-%d", RoundTo15Minutes())
		log.Printf("https://gamma-api.polymarket.com/markets/slug/%s", marketSlug)

		market, err := polymarketClient.FetchMarketBySlug(marketSlug)
		if err != nil {
			log.Fatal("FetchMarketBySlug failed:", err)
			return
		}

		endData, err := ToTimestamp(market.Get("endDate").String())
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
		pm := NewPriceManager()
		pm.SubscribeToMarket(tokenIds...)
		pm.Subscribe(func(priceData *PriceData) {
			// log.Printf("book: %+v", priceData)
			token0 := pm.GetCurrentPrice(tokenIds[0])
			token1 := pm.GetCurrentPrice(tokenIds[1])
			if token0 == nil || token1 == nil || token0.BestAsk.Price == 0 || token1.BestAsk.Price == 0 {
				return
			}
			if token0.BestAsk.Price+token1.BestAsk.Price < 1.0 {
				log.Printf("Book Data === BestAsk: %.2f/%.2f=%.2f, %.2f/%.2f", token0.BestAsk.Price, token1.BestAsk.Price, token0.BestAsk.Price+token1.BestAsk.Price, token0.BestAsk.Size, token1.BestAsk.Size)
			}
			if time.Now().UnixMilli()-endData > 1000 {
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
