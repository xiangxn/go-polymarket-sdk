package polymarket

import (
	"time"

	"github.com/xiangxn/go-polymarket-sdk/orders"
)

func (c *PolymarketConfig) HasCLOBAuth() bool {
	return c.CLOBCreds != nil
}

func (c *PolymarketConfig) HasBuilderAuth() bool {
	return c.BuilderCreds != nil
}

func (c *PolymarketConfig) HasRelayerAuth() bool {
	return c.RelayerKey != nil
}

func DefaultConfig() *Config {
	return &Config{
		HttpTimeout: 10 * time.Second,
		SocksProxy:  "",
		HttpDebug:   false,
		Polymarket: PolymarketConfig{
			ChainID:        137,
			ClobBaseURL:    "https://clob.polymarket.com",
			ClobWSBaseURL:  "wss://ws-subscriptions-clob.polymarket.com",
			LiveWSBaseURL:  "wss://ws-live-data.polymarket.com",
			GammaBaseURL:   "https://gamma-api.polymarket.com",
			RelayerBaseURL: "https://relayer-v2.polymarket.com",
			DataAPIBaseURL: "https://data-api.polymarket.com",

			SignatureType: orders.POLY_GNOSIS_SAFE,

			BuilderCode:  nil,
			OwnerKey:     "1111111111111111111111111111111111111111111111111111111111111111",
			CLOBCreds:    nil,
			BuilderCreds: nil,
		},
	}
}
