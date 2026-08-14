package polymarket

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/utils"
)

// TestCryptoPriceMonitor 订阅 Chainlink TWAP 价格:
// BTC -> 30s 窗口(btc 等价 btc_30)，BTC_60 -> 60s 窗口，ETH -> 30s 窗口
// 验证: BTC 两个窗口都能收到,ETH 只收到 30s、收不到 60s
// 运行: https_proxy=http://127.0.0.1:1087 go test -v -run TestCryptoPriceMonitor -timeout 120s ./polymarket/
func TestCryptoPriceMonitor(t *testing.T) {
	client := NewClient(DefaultConfig())
	monitor := NewCryptoPriceMonitor(client, MonitorChainlinkTwap, "BTC", "BTC_60", "ETH")

	ch := monitor.Subscribe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.Run(ctx)

	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()

	counts := map[string]map[int64]int{}
	for {
		select {
		case <-timer.C:
			for _, sym := range []string{"BTC", "ETH"} {
				log.Printf("TWAP %s: 30s=%.2f 60s=%.2f",
					sym,
					monitor.GetTwapPrice(sym, ChainlinkTwapWindowThirty),
					monitor.GetTwapPrice(sym, ChainlinkTwapWindowSixty))
			}
			check := func(symbol string, window int64, wantZero bool) {
				n := counts[symbol][window]
				if wantZero && n != 0 {
					t.Fatalf("%s %ds 不应收到价格, got %d", symbol, window, n)
				}
				if !wantZero && n == 0 {
					t.Fatalf("%s %ds 未收到价格", symbol, window)
				}
			}
			check("BTC", ChainlinkTwapWindowThirty, false)
			check("BTC", ChainlinkTwapWindowSixty, false)
			check("ETH", ChainlinkTwapWindowThirty, false)
			check("ETH", ChainlinkTwapWindowSixty, true)
			return
		case price := <-ch:
			log.Printf("price: %+v", price)
			if counts[price.Symbol] == nil {
				counts[price.Symbol] = map[int64]int{}
			}
			counts[price.Symbol][price.WindowSeconds]++
		}
	}
}

func TestBuildTwapFilters(t *testing.T) {
	f30, f60 := buildTwapFilters([]string{"BTC", "btc_60", "ETH_30", "sol_60"})
	if f30 != `[{"symbol":"btc/usd"},{"symbol":"eth/usd"}]` {
		t.Fatalf("filters30 mismatch: %s", f30)
	}
	if f60 != `[{"symbol":"btc/usd"},{"symbol":"sol/usd"}]` {
		t.Fatalf("filters60 mismatch: %s", f60)
	}

	f30, f60 = buildTwapFilters([]string{"BTC_60"})
	if f30 != "" || f60 != `[{"symbol":"btc/usd"}]` {
		t.Fatalf("single window mismatch: 30=%q 60=%q", f30, f60)
	}

	f30, f60 = buildTwapFilters(nil)
	if f30 != "" || f60 != "" {
		t.Fatalf("empty symbols mismatch: 30=%q 60=%q", f30, f60)
	}
}

func TestNewCryptoPriceMonitorTwapSubs(t *testing.T) {
	client := NewClient(DefaultConfig())

	// BTC_60 只订 60s，ETH 只订 30s
	m := NewCryptoPriceMonitor(client, MonitorChainlinkTwap, "BTC_60", "ETH")
	if len(m.subscriptions) != 2 {
		t.Fatalf("subscriptions len: %d", len(m.subscriptions))
	}
	sub30, sub60 := m.subscriptions[0], m.subscriptions[1]
	if sub30["topic"] != TOPIC_CHAINLINK_TWAP_THIRTY || sub30["filters"] != `[{"symbol":"eth/usd"}]` {
		t.Fatalf("thirty sub mismatch: %+v", sub30)
	}
	if sub60["topic"] != TOPIC_CHAINLINK_TWAP_SIXTY || sub60["filters"] != `[{"symbol":"btc/usd"}]` {
		t.Fatalf("sixty sub mismatch: %+v", sub60)
	}

	// 只订 60s 时,30s 不订阅
	m = NewCryptoPriceMonitor(client, MonitorChainlinkTwap, "BTC_60")
	if len(m.subscriptions) != 1 || m.subscriptions[0]["topic"] != TOPIC_CHAINLINK_TWAP_SIXTY {
		t.Fatalf("sixty-only subs mismatch: %+v", m.subscriptions)
	}

	// 未指定 symbol:两个窗口都订阅全部
	m = NewCryptoPriceMonitor(client, MonitorChainlinkTwap)
	if len(m.subscriptions) != 2 {
		t.Fatalf("all subs len: %d", len(m.subscriptions))
	}
}

// rawProbeHandler 打印服务器原始消息,用于诊断订阅格式
type rawProbeHandler struct {
	onOpenFn func()
}

func (h *rawProbeHandler) OnOpen() {
	log.Println("[probe] WebSocket Connected")
	if h.onOpenFn != nil {
		h.onOpenFn()
	}
}
func (h *rawProbeHandler) OnReconnect() { log.Println("[probe] WebSocket Reconnect...") }
func (h *rawProbeHandler) OnError(err error) {
	log.Println("[probe] WebSocket Error:", err)
}
func (h *rawProbeHandler) OnClose() { log.Println("[probe] WebSocket Closed") }
func (h *rawProbeHandler) OnMessage(msg []byte) {
	s := string(msg)
	if s != "PONG" && s != "" && s != "PING" {
		log.Printf("[probe] %s", s)
	}
}

// TestCryptoPriceTwapRawProbe 直接连 ws-live-data,探测 twap 订阅的 filters 格式与 payload 字段
// 运行: https_proxy=http://127.0.0.1:1087 go test -v -run TestCryptoPriceTwapRawProbe -timeout 120s ./polymarket/
func TestCryptoPriceTwapRawProbe(t *testing.T) {
	h := &rawProbeHandler{}
	ws := utils.NewWSClient(utils.WSConfig{
		URL:           "wss://ws-live-data.polymarket.com",
		PingInterval:  5 * time.Second,
		TextHeartbeat: true,
		Reconnect:     false,
	}, h)

	subMsg := `{"action":"subscribe","subscriptions":[` +
		`{"topic":"crypto_prices_twap_thirty","type":"update","filters":"[{\"symbol\":\"btc/usd\"},{\"symbol\":\"eth/usd\"}]"},` +
		`{"topic":"crypto_prices_twap_sixty","type":"update","filters":"[{\"symbol\":\"btc/usd\"},{\"symbol\":\"eth/usd\"}]"}` +
		`]}`
	h.onOpenFn = func() {
		if err := ws.Send([]byte(subMsg)); err != nil {
			log.Printf("[probe] send error: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := ws.Run(ctx); err != nil {
			log.Printf("[probe] run exit: %v", err)
		}
	}()

	time.Sleep(45 * time.Second)
}
