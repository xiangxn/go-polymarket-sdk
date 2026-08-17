package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

const SYMBOL_SUFFIX_BINANCE = "usdt"
const SYMBOL_SUFFIX_CHAINLINK = "/usd"

const TOPIC_CHAINLINK_TWAP_THIRTY = "crypto_prices_twap_thirty"
const TOPIC_CHAINLINK_TWAP_SIXTY = "crypto_prices_twap_sixty"

// Chainlink TWAP 窗口秒数
const (
	ChainlinkTwapWindowThirty = 30
	ChainlinkTwapWindowSixty  = 60
)

type MonitorType string

const (
	MonitorAll           MonitorType = "ALL"
	MonitorBinance       MonitorType = "BINANCE"
	MonitorChainlink     MonitorType = "CHAINLINK"
	MonitorChainlinkTwap MonitorType = "CHAINLINK_TWAP"
)

type ExternalPrice struct {
	Symbol        string
	Price         float64
	Source        string
	Timestamp     int64
	WindowSeconds int64 // TWAP 窗口秒数(30/60)，现货为 0
}

type CryptoPriceMonitor struct {
	ws       utils.WSClient
	pmClient *PolymarketClient

	binancePrices         map[string]float64
	chainlinkPrices       map[string]float64
	chainlinkTwap30Prices map[string]float64
	chainlinkTwap60Prices map[string]float64

	subscriptions []map[string]any

	binanceMU       sync.RWMutex
	chainlinkMU     sync.RWMutex
	chainlinkTwapMU sync.RWMutex

	priceCh chan ExternalPrice

	ctx   context.Context // Run 传入的 ctx，供 FetchOpenPrice 等 HTTP 请求继承取消/超时
	ctxMU sync.RWMutex
}

func NewCryptoPriceMonitor(pmClient *PolymarketClient, monitorType MonitorType, symbols ...string) *CryptoPriceMonitor {
	symbolBinances := utils.Map(symbols, func(symbol string) string {
		return fmt.Sprintf("{\"symbol\":\"%s%s\"}", strings.ToLower(symbol), SYMBOL_SUFFIX_BINANCE)
	})
	symbolChainlink := utils.Map(symbols, func(symbol string) string {
		return fmt.Sprintf("{\"symbol\":\"%s%s\"}", strings.ToLower(symbol), SYMBOL_SUFFIX_CHAINLINK)
	})
	sb := ""
	if len(symbolBinances) > 0 {
		sb = fmt.Sprintf("[%s]", strings.Join(symbolBinances, ","))
	}
	sc := ""
	if len(symbolChainlink) > 0 {
		sc = fmt.Sprintf("[%s]", strings.Join(symbolChainlink, ","))
	}

	cpm := CryptoPriceMonitor{
		pmClient:              pmClient,
		priceCh:               make(chan ExternalPrice, 65536),
		binancePrices:         make(map[string]float64),
		chainlinkPrices:       make(map[string]float64),
		chainlinkTwap30Prices: make(map[string]float64),
		chainlinkTwap60Prices: make(map[string]float64),
		subscriptions:         []map[string]any{},
	}
	if monitorType == MonitorAll || monitorType == MonitorBinance {
		cpm.subscriptions = append(cpm.subscriptions, map[string]any{
			"topic":   "crypto_prices",
			"type":    "update",
			"filters": sb,
		})
	}
	if monitorType == MonitorAll || monitorType == MonitorChainlink {
		cpm.subscriptions = append(cpm.subscriptions, map[string]any{
			"topic":   "crypto_prices_chainlink",
			"type":    "update",
			"filters": sc,
		})
	}
	if monitorType == MonitorAll || monitorType == MonitorChainlinkTwap {
		// 每个 topic 只发一条订阅：filters 为包含全部 symbol 的数组格式（与 chainlink 现货一致）。
		// 注意：同一 topic 的多条订阅只有一条生效，按 symbol 逐条订阅会导致只有第一个 symbol 能收到数据。
		appendTwapSub := func(topic, filters string) {
			cpm.subscriptions = append(cpm.subscriptions, map[string]any{
				"topic":   topic,
				"type":    "update",
				"filters": filters,
			})
		}
		if monitorType == MonitorChainlinkTwap {
			// MonitorChainlinkTwap 模式下 symbol 支持窗口后缀：btc 等价 btc_30（30s），btc_60 为 60s。
			// 指定了 symbol 时，某个窗口没有 symbol 就不订阅该窗口；未指定 symbol 则两个窗口都订阅全部。
			filters30, filters60 := buildTwapFilters(symbols)
			if len(symbols) == 0 {
				appendTwapSub(TOPIC_CHAINLINK_TWAP_THIRTY, "")
				appendTwapSub(TOPIC_CHAINLINK_TWAP_SIXTY, "")
			} else {
				if filters30 != "" {
					appendTwapSub(TOPIC_CHAINLINK_TWAP_THIRTY, filters30)
				}
				if filters60 != "" {
					appendTwapSub(TOPIC_CHAINLINK_TWAP_SIXTY, filters60)
				}
			}
		} else {
			appendTwapSub(TOPIC_CHAINLINK_TWAP_THIRTY, sc)
			appendTwapSub(TOPIC_CHAINLINK_TWAP_SIXTY, sc)
		}
	}

	return &cpm
}

// buildTwapFilters 解析 symbol 的窗口后缀：btc 等价 btc_30（30s），btc_60 为 60s。
// 返回 30s/60s 两个 topic 各自的 filters 数组字符串，没有 symbol 的窗口返回空串。
func buildTwapFilters(symbols []string) (string, string) {
	var s30, s60 []string
	for _, symbol := range symbols {
		lower := strings.ToLower(symbol)
		base, window := symbol, ChainlinkTwapWindowThirty
		switch {
		case strings.HasSuffix(lower, "_60"):
			base, window = symbol[:len(symbol)-3], ChainlinkTwapWindowSixty
		case strings.HasSuffix(lower, "_30"):
			base = symbol[:len(symbol)-3]
		}
		f := fmt.Sprintf("{\"symbol\":\"%s%s\"}", strings.ToLower(base), SYMBOL_SUFFIX_CHAINLINK)
		if window == ChainlinkTwapWindowSixty {
			s60 = append(s60, f)
		} else {
			s30 = append(s30, f)
		}
	}
	filters := func(list []string) string {
		if len(list) == 0 {
			return ""
		}
		return fmt.Sprintf("[%s]", strings.Join(list, ","))
	}
	return filters(s30), filters(s60)
}

func (ep *CryptoPriceMonitor) Subscribe() <-chan ExternalPrice {
	return ep.priceCh
}

func (ep *CryptoPriceMonitor) Run(ctx context.Context) error {
	log.Println("[CryptoPriceMonitor] Run start")
	defer log.Println("[CryptoPriceMonitor] Run exit")

	ep.ctxMU.Lock()
	ep.ctx = ctx
	ep.ctxMU.Unlock()

	if ep.ws != nil && ep.ws.IsAlive() {
		return nil
	}

	ep.ws = utils.NewWSClient(utils.WSConfig{
		URL:           ep.pmClient.cfg.Polymarket.LiveWSBaseURL,
		PingInterval:  5 * time.Second,
		TextHeartbeat: true, // ws-live-data 要求每 5s 发一次文本 PING
		Reconnect:     true,
		MaxReconnect:  20,
	}, ep)

	if err := ep.ws.Run(ctx); err != nil {
		return err
	}

	return ctx.Err()
}

func (ep *CryptoPriceMonitor) subscribe() {
	if ep.ws == nil || !ep.ws.IsAlive() {
		return
	}

	subscribeMessage := map[string]any{
		"action":        "subscribe",
		"subscriptions": ep.subscriptions,
	}

	data, _ := json.Marshal(subscribeMessage)
	// log.Printf("data: %s", data)
	err := ep.ws.Send(data)
	if err != nil {
		log.Printf("[CryptoPriceMonitor] 订阅标的价格失败: %v", err)
		return
	}

	log.Printf("[CryptoPriceMonitor] 📡 已订阅标的价格: %+v", ep.subscriptions)
}

func (ep *CryptoPriceMonitor) emitPrice(price ExternalPrice) {
	select {
	case ep.priceCh <- price:
	default:
		log.Println("[CryptoPriceMonitor] fill channel full, dropping fill")
	}
}

func (ep *CryptoPriceMonitor) handleMessage(msg string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[CryptoPriceMonitor] handleMessage panic: %v", r)
		}
	}()

	// 订阅后服务器会回一条 type=subscribe 的 ack（带历史数据数组，topic 仍为 crypto_prices），
	// 只有 type=update 的才是实时价格，其余一律忽略
	if gjson.Get(msg, "type").String() != "update" {
		return
	}

	topic := gjson.Get(msg, "topic").String()
	switch topic {
	case "crypto_prices":
		symbol := gjson.Get(msg, "payload.symbol").String()
		price := gjson.Get(msg, "payload.value").Float()
		timestamp := gjson.Get(msg, "payload.timestamp").Int()
		// log.Printf("Binance Price: %f %s", price, symbol)
		symbol = strings.ToUpper(strings.Replace(symbol, "usdt", "", 1))
		if symbol != "" {
			ep.binanceMU.Lock()
			ep.binancePrices[symbol] = price
			ep.binanceMU.Unlock()
			ep.emitPrice(ExternalPrice{
				Symbol:    symbol,
				Price:     price,
				Source:    "Binance",
				Timestamp: timestamp,
			})

		}
	case "crypto_prices_chainlink":
		symbol := gjson.Get(msg, "payload.symbol").String()
		price := gjson.Get(msg, "payload.value").Float()
		timestamp := gjson.Get(msg, "payload.timestamp").Int()
		// log.Printf("Chainlink Price: %f %s", price, symbol)
		symbol = strings.ToUpper(strings.Replace(symbol, "/usd", "", 1))
		if symbol != "" {
			ep.chainlinkMU.Lock()
			ep.chainlinkPrices[symbol] = price
			ep.chainlinkMU.Unlock()
			ep.emitPrice(ExternalPrice{
				Symbol:    symbol,
				Price:     price,
				Source:    "Chainlink",
				Timestamp: timestamp,
			})
		}
	case TOPIC_CHAINLINK_TWAP_THIRTY, TOPIC_CHAINLINK_TWAP_SIXTY:
		symbol := gjson.Get(msg, "payload.symbol").String()
		price := gjson.Get(msg, "payload.value").Float()
		timestamp := gjson.Get(msg, "payload.timestamp").Int()
		window := gjson.Get(msg, "payload.window_s").Int()
		// log.Printf("Chainlink TWAP Price: %f %s %ds", price, symbol, window)
		symbol = strings.ToUpper(strings.Replace(symbol, "/usd", "", 1))
		if symbol != "" && (window == ChainlinkTwapWindowThirty || window == ChainlinkTwapWindowSixty) {
			ep.chainlinkTwapMU.Lock()
			if window == ChainlinkTwapWindowThirty {
				ep.chainlinkTwap30Prices[symbol] = price
			} else {
				ep.chainlinkTwap60Prices[symbol] = price
			}
			ep.chainlinkTwapMU.Unlock()
			ep.emitPrice(ExternalPrice{
				Symbol:        symbol,
				Price:         price,
				Source:        fmt.Sprintf("ChainlinkTWAP%d", window),
				Timestamp:     timestamp,
				WindowSeconds: window,
			})
		}
	}
}

func (ep *CryptoPriceMonitor) GetExternalPrice(symbol string, resolutionSource string) float64 {
	if strings.Contains(resolutionSource, "data.chain.link") {
		ep.chainlinkMU.RLock()
		defer ep.chainlinkMU.RUnlock()
		p, ok := ep.chainlinkPrices[strings.ToUpper(symbol)]
		if !ok {
			return 0
		}
		return p
	} else if strings.Contains(resolutionSource, "www.binance.com") {
		ep.binanceMU.RLock()
		defer ep.binanceMU.RUnlock()
		p, ok := ep.binancePrices[strings.ToUpper(symbol)]
		if !ok {
			return 0
		}
		return p
	}
	return 0
}

// GetTwapPrice 返回指定 symbol 最新 Chainlink TWAP 价格，windowSeconds 为 30 或 60
func (ep *CryptoPriceMonitor) GetTwapPrice(symbol string, windowSeconds int64) float64 {
	ep.chainlinkTwapMU.RLock()
	defer ep.chainlinkTwapMU.RUnlock()
	symbol = strings.ToUpper(symbol)
	switch windowSeconds {
	case ChainlinkTwapWindowThirty:
		p, ok := ep.chainlinkTwap30Prices[symbol]
		if !ok {
			return 0
		}
		return p
	case ChainlinkTwapWindowSixty:
		p, ok := ep.chainlinkTwap60Prices[symbol]
		if !ok {
			return 0
		}
		return p
	}
	return 0
}

// return openPrice,closePrice
func (ep *CryptoPriceMonitor) FetchOpenPrice(market *gjson.Result, twapEnabled bool, twapLookbackSeconds int) (float64, float64) {
	tags := market.Get("tags").Array()
	endDate := market.Get("endDate").String()
	symbol, err := GetSymbol(tags)
	if err != nil {
		log.Printf("GetSymbol error: %v", err)
		return 0, 0
	}
	u, err := GetTimeUnit(tags)
	if err != nil {
		log.Printf("GetTimeUnit error: %v", err)
		return 0, 0
	}
	unit, err := GetSearchTimeUnit(u)
	startTime := GetStartTime(u, endDate)
	// log.Printf("symbol: %s, startTime: %s, endDate: %s, unit: %s", symbol, utils.ToISOString(startTime), utils.ToISOString(helper.TimeParse(endDate)), unit)
	ep.ctxMU.RLock()
	ctx := ep.ctx
	ep.ctxMU.RUnlock()
	if ctx == nil { // 未调用过 Run 时退化为无取消控制
		ctx = context.Background()
	}
	return ep.pmClient.FetchOpenPriceContext(ctx, symbol, startTime, utils.TimeParse(endDate), unit, twapEnabled, twapLookbackSeconds)
}

/***WSClient handler实现***/

func (ep *CryptoPriceMonitor) OnOpen() {
	log.Println("[CryptoPriceMonitor] WebSocket Connected")
	ep.subscribe()
}

func (ep *CryptoPriceMonitor) OnReconnect() {
	log.Println("[CryptoPriceMonitor] WebSocket Reconnect...")
	ep.subscribe()
}

func (ep *CryptoPriceMonitor) OnError(err error) {
	log.Println("[CryptoPriceMonitor] WebSocket Error:", err)
}

func (ep *CryptoPriceMonitor) OnClose() {
	log.Println("[CryptoPriceMonitor] WebSocket Closed")
}

func (ep *CryptoPriceMonitor) OnMessage(data []byte) {
	msg := string(data)
	if msg != "PONG" && msg != "" {
		ep.handleMessage(msg)
	}
}
