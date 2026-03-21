package tests

import (
	"testing"

	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

func TestGetOrderRawAmounts(t *testing.T) {
	t.Logf("config: %+v", orders.GetRoundConfig(orders.TickSize0001))
	size := 5 / 0.003
	side, rawMakerAmt, rawTakerAmt := orders.GetOrderRawAmounts(model.BUY, size, 0.003, orders.GetRoundConfig(orders.TickSize0001))

	t.Logf("side: %v, rawMakerAmt: %f, rawTakerAmt: %f, size: %f", side, rawMakerAmt, rawTakerAmt, size)
}

func TestGetMarketOrderRawAmounts(t *testing.T) {
	t.Logf("config: %+v", orders.GetRoundConfig(orders.TickSize0001))
	side, rawMakerAmt, rawTakerAmt := orders.GetMarketOrderRawAmounts(model.BUY, 5, 0.003, orders.GetRoundConfig(orders.TickSize0001))

	t.Logf("side: %v, rawMakerAmt: %f, rawTakerAmt: %f", side, rawMakerAmt, rawTakerAmt)
}
