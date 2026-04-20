package polymarket

import (
	"testing"
	"time"
)

func TestTradeMonitorHandleMessage_TradeAllStatusEmitted(t *testing.T) {
	tm := &TradeMonitor{eventCh: make(chan TradeEvent, 10)}

	statuses := []string{"MATCHED", "MINED", "CONFIRMED", "RETRYING", "FAILED"}
	for i, status := range statuses {
		msg := []byte(`{
			"event_type":"trade",
			"id":"trade-` + status + `",
			"market":"mkt-1",
			"asset_id":"token-1",
			"price":"0.5",
			"size":"2",
			"fee_rate_bps":"0.02",
			"side":"BUY",
			"owner":"owner-1",
			"status":"` + status + `",
			"timestamp":"171000000` + string(rune('0'+i)) + `",
			"maker_orders":[]
		}`)
		tm.handleMessage(msg)
	}

	for _, status := range statuses {
		ev := mustRecvTradeEvent(t, tm.eventCh)
		if ev.EventType != TradeEventTypeTrade {
			t.Fatalf("unexpected event type: %v", ev.EventType)
		}
		if ev.ParseErr != nil {
			t.Fatalf("unexpected parse error: %v", ev.ParseErr)
		}
		if ev.Trade == nil {
			t.Fatal("trade payload should not be nil")
		}
		if ev.Trade.Status != status {
			t.Fatalf("unexpected trade status: got=%s want=%s", ev.Trade.Status, status)
		}
	}
}

func TestTradeMonitorHandleMessage_OrderEventEmitted(t *testing.T) {
	tm := &TradeMonitor{eventCh: make(chan TradeEvent, 1)}

	msg := []byte(`{
		"event_type":"order",
		"id":"order-1",
		"market":"mkt-1",
		"asset_id":"token-1",
		"owner":"owner-1",
		"side":"BUY",
		"price":"0.42",
		"original_size":"10",
		"size_matched":"0",
		"status":"LIVE",
		"type":"PLACEMENT",
		"timestamp":"1710000010"
	}`)

	tm.handleMessage(msg)
	ev := mustRecvTradeEvent(t, tm.eventCh)

	if ev.EventType != TradeEventTypeOrder {
		t.Fatalf("unexpected event type: %v", ev.EventType)
	}
	if ev.ParseErr != nil {
		t.Fatalf("unexpected parse error: %v", ev.ParseErr)
	}
	if ev.Order == nil {
		t.Fatal("order payload should not be nil")
	}
	if ev.Order.Type != "PLACEMENT" || ev.Order.Status != "LIVE" {
		t.Fatalf("unexpected order payload: %+v", ev.Order)
	}
}

func TestTradeMonitorHandleMessage_UnknownEventEmitted(t *testing.T) {
	tm := &TradeMonitor{eventCh: make(chan TradeEvent, 1)}

	raw := []byte(`{"event_type":"book","market":"mkt-1"}`)
	tm.handleMessage(raw)
	ev := mustRecvTradeEvent(t, tm.eventCh)

	if ev.EventType != TradeEventTypeUnknown {
		t.Fatalf("unexpected event type: %v", ev.EventType)
	}
	if ev.ParseErr != nil {
		t.Fatalf("unexpected parse error: %v", ev.ParseErr)
	}
	if string(ev.Raw) != string(raw) {
		t.Fatalf("raw mismatch: got=%s want=%s", string(ev.Raw), string(raw))
	}
}

func TestTradeMonitorOnMessage_HeartbeatIgnored(t *testing.T) {
	tm := &TradeMonitor{eventCh: make(chan TradeEvent, 1)}

	tm.OnMessage([]byte("PONG"))
	tm.OnMessage([]byte("{}"))
	tm.OnMessage([]byte("   "))

	assertNoTradeEvent(t, tm.eventCh)
}

func TestTradeMonitorHandleMessage_TradeParseErrorEmitted(t *testing.T) {
	tm := &TradeMonitor{eventCh: make(chan TradeEvent, 1)}

	msg := []byte(`{
		"event_type":"trade",
		"id":"trade-err",
		"price":"bad-number"
	}`)
	tm.handleMessage(msg)
	ev := mustRecvTradeEvent(t, tm.eventCh)

	if ev.EventType != TradeEventTypeTrade {
		t.Fatalf("unexpected event type: %v", ev.EventType)
	}
	if ev.ParseErr == nil {
		t.Fatal("expected parse error")
	}
	if ev.Trade != nil {
		t.Fatal("trade payload should be nil when parse failed")
	}
}

func TestTradeMonitorHandleMessage_OrderParseErrorEmitted(t *testing.T) {
	tm := &TradeMonitor{eventCh: make(chan TradeEvent, 1)}

	msg := []byte(`{
		"event_type":"order",
		"id":"order-err",
		"price":"bad-number"
	}`)
	tm.handleMessage(msg)
	ev := mustRecvTradeEvent(t, tm.eventCh)

	if ev.EventType != TradeEventTypeOrder {
		t.Fatalf("unexpected event type: %v", ev.EventType)
	}
	if ev.ParseErr == nil {
		t.Fatal("expected parse error")
	}
	if ev.Order != nil {
		t.Fatal("order payload should be nil when parse failed")
	}
}

func mustRecvTradeEvent(t *testing.T, ch <-chan TradeEvent) TradeEvent {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for trade event")
		return TradeEvent{}
	}
}

func assertNoTradeEvent(t *testing.T, ch <-chan TradeEvent) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event emitted: %+v", ev)
	default:
	}
}
