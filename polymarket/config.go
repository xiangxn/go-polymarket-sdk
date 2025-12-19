// Package polymarket 提供与 Polymarket API 交互的功能，
package polymarket

import (
	"time"

	builderSDK "github.com/polymarket/go-builder-signing-sdk"
)

type Config struct {
	SocksProxy   *string
	Timeout      time.Duration
	CLOBCreds    *ApiKeyCreds
	BuilderCreds *builderSDK.LocalSignerConfig
}

const CollateralTokenDecimals = 6
