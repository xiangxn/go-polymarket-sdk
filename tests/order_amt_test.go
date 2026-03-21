package tests

import (
	"testing"

	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

func TestGetOrderRawAmounts(t *testing.T) {
	t.Logf("config: %+v", orders.GetRoundConfig(orders.TickSize0001))
	side, rawMakerAmt, rawTakerAmt := orders.GetOrderRawAmounts(model.BUY, 9.495, 0.55, orders.GetRoundConfig(orders.TickSize0001))

	t.Logf("side: %v, rawMakerAmt: %f, rawTakerAmt: %f", side, rawMakerAmt, rawTakerAmt)
}
