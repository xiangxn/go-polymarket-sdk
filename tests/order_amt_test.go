package tests

import (
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func TestRoundNormal(t *testing.T) {
	m := utils.RoundNormal(0.03456, 2)
	n := utils.RoundNormal(0.03556, 2)
	t.Logf("m: %.18f, m1: %s", m.InexactFloat64(), m.StringFixedBank(2))
	t.Logf("n: %.18f, n1: %s", n.InexactFloat64(), n.StringFixedBank(2))
}

func TestGetOrderRawAmounts(t *testing.T) {
	t.Logf("config: %+v", orders.GetRoundConfig(orders.TickSize001))
	size := 30.0
	side, rawMakerAmt, rawTakerAmt := orders.GetOrderRawAmounts(orders.BUY, size, 0.03, orders.GetRoundConfig(orders.TickSize001))

	t.Logf("side: %v, rawMakerAmt: %s, rawTakerAmt: %s, size: %f", side, rawMakerAmt, rawTakerAmt, size)
}

func TestGetMarketOrderRawAmounts(t *testing.T) {
	t.Logf("config: %+v", orders.GetRoundConfig(orders.TickSize0001))
	side, rawMakerAmt, rawTakerAmt := orders.GetMarketOrderRawAmounts(orders.BUY, 5, 0.003, orders.GetRoundConfig(orders.TickSize0001))

	t.Logf("side: %v, rawMakerAmt: %s, rawTakerAmt: %s", side, rawMakerAmt, rawTakerAmt)
}
