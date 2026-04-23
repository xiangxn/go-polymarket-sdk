package tests

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestServerTime(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	client := polymarket.NewClient(config)

	start := time.Now().Unix()
	serverTime, err := client.GetServerTime()
	if err != nil {
		t.Fatal(err)
	}
	end := time.Now().Unix()
	rtt := end - start
	oneWay := rtt / 2
	offset := serverTime - (start + oneWay)
	t.Logf("start: %d, serverTime: %d, end: %d, rtt: %d, oneWay: %d, offset: %d", start, serverTime, end, rtt, oneWay, offset)
}

func TestServerBookTime(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	tokenId := os.Getenv("TOKENID")
	client := polymarket.NewClient(config)

	start := time.Now().UnixMilli()
	result, err := client.GetOrderBook(tokenId)
	if err != nil {
		log.Printf("get order book error: %s", err)
	}
	bookTime := result.Timestamp
	end := time.Now().UnixMilli()

	rtt := end - start
	log.Printf("start: %d, end: %d, rtt: %d, bookTime: %d", start, end, rtt, bookTime)
	log.Printf("book - start: %d, end - book: %d", bookTime-start, end-bookTime)
}
