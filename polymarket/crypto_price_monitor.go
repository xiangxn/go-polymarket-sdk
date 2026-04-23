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

type MonitorType string

const (
	MonitorAll       MonitorType = "ALL"
	MonitorBinance   MonitorType = "BINANCE"
	MonitorChainlink MonitorType = "CHAINLINK"
)

type ExternalPrice struct {
	Symbol    string
	Price     float64
	Source    string
	Timestamp int64
}

type CryptoPriceMonitor struct {
	ws       utils.WSClient
	pmClient *PolymarketClient

	binancePrices   map[string]float64
	chainlinkPrices map[string]float64

	subscriptions []map[string]any

	binanceMU   sync.RWMutex
	chainlinkMU sync.RWMutex

	priceCh chan ExternalPrice
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
		pmClient:        pmClient,
		priceCh:         make(chan ExternalPrice, 4096),
		binancePrices:   make(map[string]float64),
		chainlinkPrices: make(map[string]float64),
		subscriptions:   []map[string]any{},
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

	return &cpm
}

func (ep *CryptoPriceMonitor) Subscribe() <-chan ExternalPrice {
	return ep.priceCh
}

func (ep *CryptoPriceMonitor) Run(ctx context.Context) error {
	log.Println("[CryptoPriceMonitor] Run start")
	defer log.Println("[CryptoPriceMonitor] Run exit")

	if ep.ws != nil && ep.ws.IsAlive() {
		return nil
	}

	ep.ws = utils.NewWSClient(utils.WSConfig{
		URL:          ep.pmClient.cfg.Polymarket.LiveWSBaseURL,
		PingInterval: 10 * time.Second,
		Reconnect:    true,
		MaxReconnect: 20,
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

func (ep *CryptoPriceMonitor) FetchOpenPrice(market *gjson.Result) float64 {
	tags := market.Get("tags").Array()
	endDate := market.Get("endDate").String()
	symbol, err := GetSymbol(tags)
	if err != nil {
		log.Print(0)
		return 0
	}
	u, err := GetTimeUnit(tags)
	if err != nil {
		log.Print(1)
		return 0
	}
	unit, err := GetSearchTimeUnit(u)
	startTime := GetStartTime(u, endDate)
	// log.Printf("symbol: %s, startTime: %s, endDate: %s, unit: %s", symbol, utils.ToISOString(startTime), utils.ToISOString(helper.TimeParse(endDate)), unit)
	return ep.pmClient.FetchOpenPrice(symbol, startTime, utils.TimeParse(endDate), unit)
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
