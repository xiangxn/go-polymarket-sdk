package tests

import (
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestGetApiKeys(t *testing.T) {
	config := polymarket.DefaultConfig()
	privateKey := os.Getenv("SIGNERKEY")

	config.Polymarket.CLOBCreds = &polymarket.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASS_PHRASE"),
	}

	client := polymarket.NewClient(privateKey, config)

	result, err := client.GetApiKeys()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}
