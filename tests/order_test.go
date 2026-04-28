package tests

import (
	"log"
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestCreateOrder(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = "95f57df83272121b4c5c43b219e6a1ab38387362e9c10c81d477accf82d84c11"
	client := polymarket.NewClient(config)

	tickSize := orders.TickSize001
	signatureType := orders.POLY_PROXY
	order, err := client.CreateOrder(&orders.UserOrder{
		TokenID: "24762431047507049460785923962525415896557183202961867581065585559228045929655",
		Price:   0.5,
		Size:    1.0,
		Side:    orders.BUY,
	}, orders.CreateOrderOptions{
		TickSize:      &tickSize,
		SignatureType: &signatureType,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("order: %+v", order)
}

func TestCreateMarketOrder(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	client := polymarket.NewClient(config)

	tokenID := os.Getenv("TOKENID")

	tickSize := orders.TickSize001
	signatureType := orders.POLY_GNOSIS_SAFE
	order, err := client.CreateMarketOrder(&orders.UserMarketOrder{
		TokenID:   tokenID,
		Amount:    1,
		Side:      orders.BUY,
		OrderType: orders.MARKET_FOK,
	}, orders.CreateOrderOptions{
		TickSize:      &tickSize,
		SignatureType: &signatureType,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("order: %+v", order)
}

// 这个测试会真实下单，请务必慎重
func TestPlaceOrder(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	funderAddress := os.Getenv("FUNDERADDRESS")
	tokenID := os.Getenv("TOKENID2")

	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}
	config.Polymarket.FunderAddress = funderAddress

	client := polymarket.NewClient(config)

	tickSize := orders.TickSize001
	signatureType := orders.POLY_GNOSIS_SAFE
	order, err := client.CreateOrder(&orders.UserOrder{
		TokenID: tokenID,
		Price:   0.2,
		Size:    5.0,
		Side:    orders.BUY,
	}, orders.CreateOrderOptions{
		TickSize:      &tickSize,
		SignatureType: &signatureType,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("order: %+v", order)

	result, err := client.PostOrder(order, orders.GTC, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}

func TestPlaceOrders(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	funderAddress := os.Getenv("FUNDERADDRESS")
	tokenID := os.Getenv("TOKENID2")

	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}
	config.Polymarket.FunderAddress = funderAddress

	client := polymarket.NewClient(config)

	tickSize := orders.TickSize001
	signatureType := orders.POLY_GNOSIS_SAFE
	order, err := client.CreateOrder(&orders.UserOrder{
		TokenID: tokenID,
		Price:   0.2,
		Size:    5.0,
		Side:    orders.BUY,
	}, orders.CreateOrderOptions{
		TickSize:      &tickSize,
		SignatureType: &signatureType,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("order: %+v", order)

	order2, err := client.CreateOrder(&orders.UserOrder{
		TokenID: tokenID,
		Price:   0.2,
		Size:    5.0,
		Side:    orders.BUY,
	}, orders.CreateOrderOptions{
		TickSize:      &tickSize,
		SignatureType: &signatureType,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("order2: %+v", order)

	var os []orders.PostOrdersArgs
	os = append(os, orders.PostOrdersArgs{
		Order:     order,
		OrderType: orders.GTC,
	})
	os = append(os, orders.PostOrdersArgs{
		Order:     order2,
		OrderType: orders.GTC,
	})
	result, err := client.PostOrders(os, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}

func TestCancelOrder(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	orderID := os.Getenv("ORDERID")
	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}
	client := polymarket.NewClient(config)

	log.Println("orderID: ", orderID)
	result, err := client.CancelOrder(&orders.OrderPayload{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}

func TestCancelOrders(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	orderID := os.Getenv("ORDERID")
	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}
	client := polymarket.NewClient(config)

	log.Println("orderID: ", orderID)
	result, err := client.CancelOrders([]string{orderID})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}

func TestPlaceMarketOrder(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}
	funderAddress := os.Getenv("FUNDERADDRESS")
	config.Polymarket.FunderAddress = funderAddress

	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	client := polymarket.NewClient(config)

	tokenID := os.Getenv("TOKENID")

	tickSize := orders.TickSize001
	signatureType := orders.POLY_GNOSIS_SAFE
	order, err := client.CreateMarketOrder(&orders.UserMarketOrder{
		TokenID:   tokenID,
		Amount:    1,
		Side:      orders.BUY,
		OrderType: orders.MARKET_FOK,
	}, orders.CreateOrderOptions{
		TickSize:      &tickSize,
		SignatureType: &signatureType,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("order: %+v", order)

	result, err := client.PostOrder(order, orders.FOK, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}

func TestGetOpenOrders(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}

	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	client := polymarket.NewClient(config)

	result, err := client.GetOpenOrders(nil, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}
