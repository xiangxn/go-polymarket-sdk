package polymarket

import (
	"math"
	"testing"
	"time"

	pmModel "github.com/polymarket/go-order-utils/pkg/model"
	sdkModel "github.com/xiangxn/go-polymarket-sdk/model"
)

func TestTradeMonitorHandleMessage_EmitsTakerAndMakerFills(t *testing.T) {
	creds := &sdkModel.ApiKeyCreds{
		Key:        "614db9ff-874b-581b-60b3-e264e5fa4802",
		Secret:     "",
		Passphrase: "",
	}
	tm := &TradeMonitor{
		subscribeTradeStatus: "",
		fillCh:               make(chan Fill, 10),
		creds:                creds,
	}

	msg := []byte(`{
		"event_type":"trade",
		"id":"fill-1",
		"market":"mkt-1",
		"matchtime":"1710000000",
		"owner":"614db9ff-874b-581b-60b3-e264e5fa4802",
		"side":"SELL",
		"asset_id":"token-taker",
		"price":"0.4",
		"size":"5",
		"fee_rate_bps":"0.02",
		"taker_order_id":"order-taker",
		"maker_orders":[
			{
				"owner":"614db9ff-874b-581b-60b3-e264e5fa4802",
				"side":"BUY",
				"order_id":"order-maker-1",
				"asset_id":"token-maker-1",
				"price":"0.35",
				"matched_amount":"2",
				"fee_rate_bps":"0.01"
			},
			{
				"owner":"0xdef",
				"side":"SELL",
				"order_id":"order-maker-2",
				"asset_id":"token-maker-2",
				"price":"0.9",
				"matched_amount":"1",
				"fee_rate_bps":"0.03"
			}
		]
	}`)

	tm.handleMessage(msg)

	f1 := mustRecvFill(t, tm.fillCh)
	f2 := mustRecvFill(t, tm.fillCh)

	if f1.OrderID != "order-taker" || f1.TokenID != "token-taker" || f1.FillID != "fill-1" || f1.MarketID != "mkt-1" {
		t.Fatalf("unexpected taker fill: %+v", f1)
	}
	if f1.Side != pmModel.SELL {
		t.Fatalf("expected taker side SELL, got %v", f1.Side)
	}
	assertFloatEqual(t, f1.Price, 0.4)
	assertFloatEqual(t, f1.Size, 5)
	assertFloatEqual(t, f1.Fee, 0.02*5*0.4)

	if f2.OrderID != "order-maker-1" || f2.TokenID != "token-maker-1" {
		t.Fatalf("unexpected maker fill: %+v", f2)
	}
	if f2.Side != pmModel.BUY {
		t.Fatalf("expected maker side BUY, got %v", f2.Side)
	}
	assertFloatEqual(t, f2.Price, 0.35)
	assertFloatEqual(t, f2.Size, 2)
	assertFloatEqual(t, f2.Fee, 0.01*2*0.35)

	select {
	case extra := <-tm.fillCh:
		t.Fatalf("unexpected extra fill emitted: %+v", extra)
	default:
	}
}

func TestTradeMonitorHandleMessage_NonTradeIgnored(t *testing.T) {
	tm := &TradeMonitor{fillCh: make(chan Fill, 1)}
	tm.handleMessage([]byte(`{"event_type":"book"}`))

	select {
	case f := <-tm.fillCh:
		t.Fatalf("unexpected fill emitted: %+v", f)
	default:
	}
}

func mustRecvFill(t *testing.T, ch <-chan Fill) Fill {
	t.Helper()
	select {
	case f := <-ch:
		return f
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for fill")
		return Fill{}
	}
}

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("float mismatch, got=%v want=%v", got, want)
	}
}
