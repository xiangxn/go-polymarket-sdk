package tests

import (
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestServerTime(t *testing.T) {
	config := polymarket.DefaultConfig()
	privateKey := os.Getenv("SIGNERKEY")
	client := polymarket.NewClient(privateKey, config)

	serverTime, err := client.GetServerTime()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("serverTime: %+v", serverTime)
}
