package polymarket

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type ContractConfig struct {
	Exchange          common.Address
	NegRiskAdapter    common.Address
	NegRiskExchange   common.Address
	Collateral        common.Address
	ConditionalTokens common.Address
}

type Signer struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}

// ----------------------
// 数据结构定义
// ----------------------

type OrderBookSummary struct {
	Market       string        `json:"market"`
	AssetId      string        `json:"asset_id"`
	Timestamp    int64         `json:"timestamp"`
	Bids         []orders.Book `json:"bids"`
	Asks         []orders.Book `json:"asks"`
	MinOrderSize float64       `json:"min_order_size"`
	TickSize     float64       `json:"tick_size"`
	NegRisk      bool          `json:"neg_risk"`
	Hash         string        `json:"hash"`
}

type OrderBook struct {
	Market  string `json:"market"`
	AssetId string `json:"asset_id"`
	// 触发时间戳，毫秒
	Timestamp int64 `json:"timestamp"`
	// 接收延迟，毫秒
	Latency int64         `json:"latency,omitempty"`
	Bids    []orders.Book `json:"bids"`
	Asks    []orders.Book `json:"asks"`
}

type BookParams struct {
	TokenId string       `json:"token_id"`
	Side    *orders.Side `json:"side,omitempty"`
}

// PriceData 表示价格数据,这个数据中的(MinOrderSize,TickSize,NegRisk)只在rest api中返回,ws api不返回
type PriceData struct {
	TokenID   string       `json:"tokenId"`
	Market    string       `json:"market"`
	BestAsk   *orders.Book `json:"bestAsk,omitempty"`
	BestBid   *orders.Book `json:"bestBid,omitempty"`
	Timestamp int64        `json:"timestamp"`

	MinOrderSize float64 `json:"min_order_size"`
	TickSize     float64 `json:"tick_size"`
	NegRisk      bool    `json:"neg_risk"`
}

// PriceUpdateCallback 价格更新回调函数类型
type PriceUpdateCallback func(priceData *PriceData)

// MarketMessage 市场订阅消息
type MarketMessage struct {
	Type      string   `json:"type"`
	AssetsIDs []string `json:"assets_ids"`
}

// WSMessage WebSocket消息
type WSMessage struct {
	EventType string        `json:"event_type"`
	Market    string        `json:"market"`
	AssetID   string        `json:"asset_id"`
	Bids      []orders.Book `json:"bids"`
	Asks      []orders.Book `json:"asks"`
}

type CryptoPriceSymbol string

const (
	SOL CryptoPriceSymbol = "SOL"
	BTC CryptoPriceSymbol = "BTC"
	ETH CryptoPriceSymbol = "ETH"
	XRP CryptoPriceSymbol = "XRP"
)

type CryptoPriceUint string

const (
	Fifteen  CryptoPriceUint = "fifteen"
	Hourly   CryptoPriceUint = "hourly"
	Fourhour CryptoPriceUint = "fourhour"
	Daily    CryptoPriceUint = "daily"
	Weekly   CryptoPriceUint = "weekly"
	Monthly  CryptoPriceUint = "monthly"
)
