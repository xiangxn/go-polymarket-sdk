package tests

import (
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestGetApiKeys(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")

	config.Polymarket.CLOBCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("CLOB_API_KEY"),
		Secret:     os.Getenv("CLOB_SECRET"),
		Passphrase: os.Getenv("CLOB_PASSPHRASE"),
	}

	client := polymarket.NewClient(config)

	result, err := client.GetApiKeys()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}
