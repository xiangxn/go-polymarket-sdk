package polymarket

import (
	"time"
)

func (c *PolymarketConfig) HasCLOBAuth() bool {
	return c.CLOBCreds != nil
}

func (c *PolymarketConfig) HasBuilderAuth() bool {
	return c.BuilderCreds != nil
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
			GammaBaseURL:   "https://gamma-api.polymarket.com",
			RelayerBaseURL: "https://relayer-v2.polymarket.com",
			DataAPIBaseURL: "https://data-api.polymarket.com",
			CLOBCreds:      nil,
			BuilderCreds:   nil,
		},
	}
}
