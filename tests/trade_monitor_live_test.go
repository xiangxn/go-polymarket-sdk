package tests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkModel "github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestTradeMonitorLive_FromDotEnv(t *testing.T) {
	loadDotEnv()

	wsURL := getEnvOrDefault("CLOB_WS_BASE_URL", "wss://ws-subscriptions-clob.polymarket.com")
	funder := strings.TrimSpace(os.Getenv("FUNDERADDRESS"))
	if funder == "" {
		t.Skip("skip live test: FUNDERADDRESS is empty")
	}

	t.Logf("funder: %s", funder)

	creds := &sdkModel.ApiKeyCreds{
		Key:        strings.TrimSpace(os.Getenv("CLOB_API_KEY")),
		Secret:     strings.TrimSpace(os.Getenv("CLOB_SECRET")),
		Passphrase: strings.TrimSpace(os.Getenv("CLOB_PASSPHRASE")),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Skip("skip live test: CLOB_API_KEY/CLOB_SECRET/CLOB_PASSPHRASE is not fully set")
	}

	tm := polymarket.NewTradeMonitor(wsURL, creds)

	ctx, cancel := context.WithTimeout(context.Background(), 480*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- tm.Run(ctx)
	}()
	defer tm.Close()

	eventCount := 0
	tradeCount := 0
	orderCount := 0
	unknownCount := 0
	fillCount := 0
	ownOrders := make(map[string]struct{})
	for {
		select {
		case ev := <-tm.SubscribeEvents():
			eventCount++
			switch ev.EventType {
			case polymarket.TradeEventTypeTrade:
				tradeCount++
			case polymarket.TradeEventTypeOrder:
				orderCount++
			default:
				unknownCount++
			}
			fills := buildOwnOrderFillsFromTradeEvent(ev, ownOrders)
			fillCount += len(fills)
			t.Logf("got live event #%d type=%s parseErr=%v ownDerivedFills=%d", eventCount, ev.EventType, ev.ParseErr, len(fills))
			for i, f := range fills {
				t.Logf("  own fill[%d]: %+v action=%s", i, f, suggestActionByStatus(f.Status))
			}
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("trade monitor run failed: %v", err)
			}
			t.Logf("trade monitor exited early, eventCount=%d trade=%d order=%d unknown=%d fill=%d err=%v", eventCount, tradeCount, orderCount, unknownCount, fillCount, err)
			return
		case <-ctx.Done():
			err := <-errCh
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("trade monitor run failed: %v", err)
			}
			t.Logf("listen finished after 480s, eventCount=%d trade=%d order=%d unknown=%d fill=%d", eventCount, tradeCount, orderCount, unknownCount, fillCount)
			return
		}
	}
}

type Fill struct {
	EventType string
	Role      string
	FillID    string
	OrderID   string
	MarketID  string
	TokenID   string
	Status    string
	Side      string
	Price     float64
	Size      float64
	Fee       float64
	Time      int64
}

func buildOwnOrderFillsFromTradeEvent(ev polymarket.TradeEvent, ownOrders map[string]struct{}) []Fill {
	switch ev.EventType {
	case polymarket.TradeEventTypeOrder:
		if ev.Order == nil {
			return nil
		}
		if ev.Order.Id != "" {
			ownOrders[ev.Order.Id] = struct{}{}
		}
		return []Fill{{
			EventType: string(ev.EventType),
			Role:      "order",
			FillID:    ev.Order.Id,
			OrderID:   ev.Order.Id,
			MarketID:  ev.Order.Market,
			TokenID:   ev.Order.AssetId,
			Status:    ev.Order.Status,
			Side:      ev.Order.Side,
			Price:     ev.Order.Price,
			Size:      ev.Order.OriginalSize,
			Fee:       0,
			Time:      ev.Order.Timestamp,
		}}
	case polymarket.TradeEventTypeTrade:
		if ev.Trade == nil {
			return nil
		}
		fills := make([]Fill, 0, 1+len(ev.Trade.MakerOrders))
		if _, ok := ownOrders[ev.Trade.TakerOrderId]; ok {
			fills = append(fills, Fill{
				EventType: string(ev.EventType),
				Role:      "taker",
				FillID:    ev.Trade.Id,
				OrderID:   ev.Trade.TakerOrderId,
				MarketID:  ev.Trade.Market,
				TokenID:   ev.Trade.AssetId,
				Status:    ev.Trade.Status,
				Side:      ev.Trade.Side,
				Price:     ev.Trade.Price,
				Size:      ev.Trade.Size,
				Fee:       ev.Trade.FeeRateBps * ev.Trade.Size * ev.Trade.Price,
				Time:      ev.Trade.Timestamp,
			})
		}
		for _, mo := range ev.Trade.MakerOrders {
			if _, ok := ownOrders[mo.OrderId]; !ok {
				continue
			}
			fills = append(fills, Fill{
				EventType: string(ev.EventType),
				Role:      "maker",
				FillID:    ev.Trade.Id,
				OrderID:   mo.OrderId,
				MarketID:  ev.Trade.Market,
				TokenID:   mo.AssetId,
				Status:    ev.Trade.Status,
				Side:      mo.Side,
				Price:     mo.Price,
				Size:      mo.MatchedAmount,
				Fee:       mo.FeeRateBps * mo.MatchedAmount * mo.Price,
				Time:      ev.Trade.Timestamp,
			})
		}
		return fills
	default:
		return nil
	}
}

func suggestActionByStatus(status string) string {
	switch strings.ToUpper(status) {
	case "MATCHED":
		return "已撮合(链下)，可更新预成交/冻结变动"
	case "MINED":
		return "已上链，可作为成交确认并更新持仓"
	case "CONFIRMED":
		return "链上确认完成，可做最终入账/风控结算"
	case "LIVE":
		return "订单仍在簿上等待撮合"
	case "CANCELED":
		return "订单已取消，可释放剩余挂单占用"
	case "FAILED":
		return "链上或执行失败，建议回滚临时状态并告警"
	default:
		return "按业务策略处理"
	}
}

func getEnvOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func loadDotEnv() {
	paths := []string{".env", filepath.Join("..", ".env")}
	for _, p := range paths {
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for line := range strings.SplitSeq(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)
			if k != "" {
				_, exists := os.LookupEnv(k)
				if !exists {
					_ = os.Setenv(k, v)
				}
			}
		}
		return
	}
}
