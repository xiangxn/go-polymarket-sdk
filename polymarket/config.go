// Package polymarket 提供与 Polymarket API 交互的功能，
package polymarket

import (
	"math/big"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/headers"
)

type PolymarketConfig struct {
	ChainID        *big.Int `mapstructure:"chain_id"`
	FunderAddress  *string  `mapstructure:"funder_address"`
	ClobBaseURL    string   `mapstructure:"clob_base_url"`
	ClobWSBaseURL  string   `mapstructure:"clob_ws_base_url"`
	GammaBaseURL   string   `mapstructure:"gamma_base_url"`
	RelayerBaseURL string   `mapstructure:"relayer_base_url"`
	DataAPIBaseURL string   `mapstructure:"data_api_base_url"`

	CLOBCreds    *headers.ApiKeyCreds `mapstructure:"clob_creds"`
	BuilderCreds *headers.ApiKeyCreds `mapstructure:"builder_creds"`
}

type Config struct {
	HttpTimeout time.Duration `mapstructure:"http_timeout"`
	SocksProxy  string        `mapstructure:"socks_proxy"`

	Polymarket PolymarketConfig `mapstructure:"polymarket"`
}
