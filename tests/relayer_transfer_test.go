package tests

import (
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/headers"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/relayer"
)

func TestTransfer(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	erc20 := os.Getenv("ERC20_USDC")

	config.Polymarket.RelayerKey = &headers.RelayerKey{
		ApiKey:        os.Getenv("RELAYER_API_KEY"),
		ApiKeyAddress: os.Getenv("RELAYER_API_KEY_ADDRESS"),
	}

	relayClient := relayer.NewRelayClient(config.Polymarket.RelayerBaseURL, config.Polymarket.OwnerKey, 137, config.Polymarket.BuilderCreds, nil, config.Polymarket.RelayerKey)

	transferResult, err := relayClient.Transfer(erc20, config.Polymarket.RelayerKey.ApiKeyAddress, "9.88", 6)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("transfer result: %+v", transferResult)
}
