package tests

import (
	"math/big"
	"testing"

	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestCreateOrder(t *testing.T) {
	config := polymarket.DefaultConfig()
	client := polymarket.NewClient("95f57df83272121b4c5c43b219e6a1ab38387362e9c10c81d477accf82d84c11", config)

	order, err := client.CreateOrder(polymarket.UserOrder{
		TokenID: "24762431047507049460785923962525415896557183202961867581065585559228045929655",
		Price:   0.5,
		Size:    1.0,
		Side:    model.BUY,
	}, polymarket.CreateOrderOptions{
		TickSize:      polymarket.TickSize001,
		SignatureType: model.POLY_PROXY,
		ChainID:       big.NewInt(137),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("order: %+v", order)
}
