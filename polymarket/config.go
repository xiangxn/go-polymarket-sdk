// Package polymarket 提供与 Polymarket API 交互的功能，
package polymarket

import (
	"time"

	builderSDK "github.com/polymarket/go-builder-signing-sdk"
	"github.com/xiangxn/go-polymarket-sdk/headers"
)

type PolymarketConfig struct {
	ClobBaseURL    string
	ClobWSBaseSURL string
	GammaBaseURL   string
	RelayerBaseURL string
	DataAPIBaseURL string

	CLOBCreds    *headers.ApiKeyCreds
	BuilderCreds *builderSDK.LocalSignerConfig
}

type Config struct {
	HttpTimeout time.Duration
	SocksProxy  string

	Polymarket PolymarketConfig
}
