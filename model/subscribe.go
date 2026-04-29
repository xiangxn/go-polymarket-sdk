package model

import (
	"strconv"
	"strings"
)

type SubscribeUserMessage struct {
	Type    string      `json:"type"`
	Markets []string    `json:"markets"`
	Auth    *WSUserAuth `json:"auth,omitempty"`
}

type WSUserAuth struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

type ApiKeyCreds struct {
	Key        string `mapstructure:"key" json:"key"`
	Secret     string `mapstructure:"secret" json:"secret"`
	Passphrase string `mapstructure:"passphrase" json:"passphrase"`
}

type WSOrder struct {
	AssetId         string   `json:"asset_id"`
	AssociateTrades []string `json:"associate_trades"`
	EventType       string   `json:"event_type"`
	Id              string   `json:"id"`
	Market          string   `json:"market"`
	OrderOwner      string   `json:"order_owner"`
	OriginalSize    float64  `json:"original_size,string"`
	Outcome         string   `json:"outcome"`
	Owner           string   `json:"owner"`
	Price           float64  `json:"price,string"`
	Side            string   `json:"side"`
	SizeMatched     float64  `json:"size_matched,string"`
	Timestamp       int64    `json:"timestamp,string"`
	Type            string   `json:"type"`
	// 'LIVE', 'MATCHED', 'CANCELED'
	Status string `json:"status"`
}

type WSMakerOrder struct {
	AssetId       string      `json:"asset_id"`
	MatchedAmount float64     `json:"matched_amount,string"`
	OrderId       string      `json:"order_id"`
	Outcome       string      `json:"outcome"`
	Owner         string      `json:"owner"`
	Side          string      `json:"side"`
	Price         float64     `json:"price,string"`
	FeeRateBps    SafeFloat64 `json:"fee_rate_bps,string"`
}

type WSTrade struct {
	AssetId     string         `json:"asset_id"`
	EventType   string         `json:"event_type"`
	Id          string         `json:"id"`
	LastUpdate  int64          `json:"last_update,string"`
	MakerOrders []WSMakerOrder `json:"maker_orders"`
	Market      string         `json:"market"`
	Matchtime   int64          `json:"matchtime,string"`
	Outcome     string         `json:"outcome"`
	Owner       string         `json:"owner"`
	Price       float64        `json:"price,string"`
	Side        string         `json:"side"`
	Size        float64        `json:"size,string"`
	FeeRateBps  SafeFloat64    `json:"fee_rate_bps,string"`
	// MATCHED, MINED, CONFIRMED, RETRYING, FAILED
	Status       string `json:"status"`
	TakerOrderId string `json:"taker_order_id"`
	Timestamp    int64  `json:"timestamp,string"`
	TradeOwner   string `json:"trade_owner"`
	Type         string `json:"type"`
}

type SafeFloat64 float64

func (f *SafeFloat64) UnmarshalJSON(b []byte) error {
	str := strings.Trim(string(b), `"`)

	if str == "" || str == "null" {
		*f = 0
		return nil
	}

	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}

	*f = SafeFloat64(v)
	return nil
}
