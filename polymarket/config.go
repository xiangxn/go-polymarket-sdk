// Package polymarket 提供与 Polymarket API 交互的功能，
package polymarket

import (
	"time"

	"github.com/xiangxn/go-polymarket-sdk/model"
)

type PolymarketConfig struct {
	ChainID        int64  `mapstructure:"chain_id"`
	FunderAddress  string `mapstructure:"funder_address"`
	ClobBaseURL    string `mapstructure:"clob_base_url"`
	ClobWSBaseURL  string `mapstructure:"clob_ws_base_url"`
	LiveWSBaseURL  string `mapstructure:"live_ws_base_url"`
	GammaBaseURL   string `mapstructure:"gamma_base_url"`
	RelayerBaseURL string `mapstructure:"relayer_base_url"`
	DataAPIBaseURL string `mapstructure:"data_api_base_url"`

	CLOBCreds    *model.ApiKeyCreds `mapstructure:"clob_creds"`
	BuilderCreds *model.ApiKeyCreds `mapstructure:"builder_creds"`
}

type Config struct {
	HttpTimeout time.Duration `mapstructure:"http_timeout"`
	SocksProxy  string        `mapstructure:"socks_proxy"`
	HttpDebug   bool          `mapstructure:"http_debug"`

	Polymarket PolymarketConfig `mapstructure:"polymarket"`
}
