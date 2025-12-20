package tests

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestGetOrderBook(t *testing.T) {
	config := polymarket.DefaultConfig()
	privateKey := os.Getenv("SIGNERKEY")
	client := polymarket.NewClient(privateKey, config)

	tokenID := os.Getenv("TOKENID")
	orderBook, err := client.GetOrderBook(tokenID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.MarshalIndent(orderBook, "", "  ")
	t.Logf("orderBook: %s", data)
}

func TestGetOrderBooks(t *testing.T) {
	config := polymarket.DefaultConfig()
	privateKey := os.Getenv("SIGNERKEY")
	tokenID := os.Getenv("TOKENID")
	tokenID2 := os.Getenv("TOKENID2")
	client := polymarket.NewClient(privateKey, config)

	orderBooks, err := client.GetOrderBooks([]polymarket.BookParams{{
		TokenId: tokenID,
		Side:    polymarket.BUY,
	}, {
		TokenId: tokenID2,
		Side:    polymarket.BUY,
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.MarshalIndent(orderBooks, "", "  ")
	t.Logf("orderBooks: %s", data)
}
