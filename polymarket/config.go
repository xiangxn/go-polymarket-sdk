// Package polymarket 提供与 Polymarket API 交互的功能，
package polymarket

import (
	"time"

	"github.com/xiangxn/go-polymarket-sdk/headers"
	"github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
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

	SignatureType orders.SignatureType `mapstructure:"signature_type"`

	// 加密价格接口地址，为空时使用默认 https://polymarket.com/api/crypto/crypto-price
	CryptoPriceURL string `mapstructure:"crypto_price_url"`

	OwnerKey     string              `mapstructure:"owner_key"`
	BuilderCode  *string             `mapstructure:"builder_code"`
	CLOBCreds    *model.ApiKeyCreds  `mapstructure:"clob_creds"`
	BuilderCreds *model.ApiKeyCreds  `mapstructure:"builder_creds"`
	RelayerKey   *headers.RelayerKey `mapstructure:"relayer_key"`
}

type Config struct {
	HttpTimeout time.Duration `mapstructure:"http_timeout"`
	SocksProxy  string        `mapstructure:"socks_proxy"`
	HttpDebug   bool          `mapstructure:"http_debug"`

	// 429 限流重试配置，0 使用默认值（3 次重试、500ms 基础退避）
	RateLimitMaxRetries int           `mapstructure:"rate_limit_max_retries"`
	RateLimitBaseDelay  time.Duration `mapstructure:"rate_limit_base_delay"`

	Polymarket PolymarketConfig `mapstructure:"polymarket"`
}
