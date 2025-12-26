// Package polymarket 提供与 Polymarket API 交互的功能，
package polymarket

import (
	"time"

	"github.com/xiangxn/go-polymarket-sdk/headers"
)

type PolymarketConfig struct {
	ClobBaseURL    string `json:"clob_base_url"`
	ClobWSBaseURL  string `json:"clob_ws_base_url"`
	GammaBaseURL   string `json:"gamma_base_url"`
	RelayerBaseURL string `json:"relayer_base_url"`
	DataAPIBaseURL string `json:"data_api_base_url"`

	CLOBCreds    *headers.ApiKeyCreds `json:"clob_creds"`
	BuilderCreds *headers.ApiKeyCreds `json:"builder_creds"`
}

type Config struct {
	HttpTimeout time.Duration `json:"http_timeout"`
	SocksProxy  string        `json:"socks_proxy"`

	Polymarket PolymarketConfig `json:"polymarket"`
}
