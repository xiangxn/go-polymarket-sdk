package polymarket

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type OperationType = string

const (
	DynamicSub   OperationType = "subscribe"
	DynamicUnSub OperationType = "unsubscribe"
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
	Type                 string   `json:"type"`
	AssetsIDs            []string `json:"assets_ids"`
	CustomFeatureEnabled bool     `json:"custom_feature_enabled"`
	InitialDump          bool     `json:"initial_dump"`
}

type DynamicSubMarketMessage struct {
	AssetsIds            []string      `json:"assets_ids"`
	Operation            OperationType `json:"operation"`
	CustomFeatureEnabled *bool         `json:"custom_feature_enabled,omitempty"`
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
	Fiveminute CryptoPriceUint = "fiveminute"
	Fifteen    CryptoPriceUint = "fifteen"
	Hourly     CryptoPriceUint = "hourly"
	Fourhour   CryptoPriceUint = "fourhour"
	Daily      CryptoPriceUint = "daily"
	Weekly     CryptoPriceUint = "weekly"
	Monthly    CryptoPriceUint = "monthly"
)

type ResolvedInfo struct {
	EventType      string   `json:"event_type"`
	Id             string   `json:"id"`
	Market         string   `json:"market"`
	AssetsIds      []string `json:"assets_ids"`
	WinningAssetId string   `json:"winning_asset_id"`
	WinningOutcome string   `json:"winning_outcome"`
	Timestamp      int64    `json:"timestamp"`
	Tags           []string `json:"tags"`
}

// PriceChangeItem 单笔价格变化
type PriceChangeItem struct {
	AssetID string `json:"asset_id"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Side    string `json:"side"`
	Hash    string `json:"hash"`
	BestBid string `json:"best_bid"`
	BestAsk string `json:"best_ask"`
}

// PriceChangeInfo price_change 事件
type PriceChangeInfo struct {
	EventType    string            `json:"event_type"`
	Market       string            `json:"market"`
	PriceChanges []PriceChangeItem `json:"price_changes"`
	// 触发时间戳，毫秒
	Timestamp int64 `json:"timestamp"`
}

// LastTradePriceInfo last_trade_price 事件
type LastTradePriceInfo struct {
	EventType       string `json:"event_type"`
	AssetID         string `json:"asset_id"`
	Market          string `json:"market"`
	Price           string `json:"price"`
	Size            string `json:"size"`
	FeeRateBps      string `json:"fee_rate_bps"`
	Side            string `json:"side"`
	TransactionHash string `json:"transaction_hash"`
	// 触发时间戳，毫秒
	Timestamp int64 `json:"timestamp"`
}

// ----------------------
// Data API v2：/v2/positions
// ----------------------

// PositionStatus /v2/positions 的 status 过滤值
type PositionStatus string

const (
	// PositionOpen 未平仓，含已结算未赎回的赢单，为服务端默认值
	PositionOpen PositionStatus = "OPEN"
	// PositionRedeemable 自 OPEN 收窄到已结算且仍持有、可立即赎回的仓位
	PositionRedeemable PositionStatus = "REDEEMABLE"
	// PositionRedeemableLost 仍持有且已归零的输单，需配合 User 使用
	PositionRedeemableLost PositionStatus = "REDEEMABLE_LOST"
	// PositionMergeable 自 OPEN 收窄到持有互补两条腿、可 merge 回抵押品的仓位
	PositionMergeable PositionStatus = "MERGEABLE"
	// PositionClosed 已退出的仓位
	PositionClosed PositionStatus = "CLOSED"
)

// PositionFilterType 过滤下限的计量方式
type PositionFilterType string

const (
	// PositionFilterTokens 按当前持仓份额过滤，为服务端默认值
	PositionFilterTokens PositionFilterType = "TOKENS"
	// PositionFilterCash 按市值(USDC)过滤
	PositionFilterCash PositionFilterType = "CASH"
)

// PositionParams /v2/positions 查询参数，除 User/Condition 至少填一个外均为可选
type PositionParams struct {
	// User 钱包地址，留空时由 SearchPositions 回落到配置的 FunderAddress
	User string
	// Condition 条件 id，逗号分隔最多 20 个
	Condition string
	// Status 仓位状态过滤，留空时服务端按 OPEN 处理
	Status PositionStatus
	// EventId 事件 id，逗号分隔最多 20 个，仅 user 维度可用
	EventId string
	// Title 市场标题子串，大小写不敏感
	Title string
	// FilterType 过滤下限的计量方式，留空时服务端按 TOKENS 处理
	FilterType PositionFilterType
	// FilterAmount 过滤下限：TOKENS 为当前持仓份额，CASH 为市值(USDC)。
	// 注意 v2 固定带 0.1 股 dust 下限，传 0 并不会解除
	FilterAmount *float64
	// SortBy 取值 CURRENT_VALUE/PRICE/TOKENS/UNREALIZED_PNL/REALIZED_PNL/TOTAL_PNL/TIMESTAMP，
	// 服务端按 status 取默认值（OPEN/REDEEMABLE 为 CURRENT_VALUE，CLOSED 为 REALIZED_PNL）
	SortBy string
	// SortDirection ASC 或 DESC
	SortDirection string
	// Limit 单页条数，上限 1000；翻页时由 cursor 接管，该值被忽略
	Limit int
}

// Position /v2/positions 单行，字段与 v2 响应一一对应
// （v1 的 camelCase 字段在 v2 已全部改名，如 conditionId→condition_id、size→current_size）
type Position struct {
	ProxyWallet        string  `json:"proxy_wallet"`
	TokenId            string  `json:"token_id"`
	ConditionId        string  `json:"condition_id"`
	CurrentSize        float64 `json:"current_size"` // 当前持仓份额，对应 v1 的 size
	AvgPrice           float64 `json:"avg_price"`
	EntryCostUsdc      float64 `json:"entry_cost_usdc"`
	EntryFeesUsdc      float64 `json:"entry_fees_usdc"`
	TotalCostUsdc      float64 `json:"total_cost_usdc"`
	CurrentPrice       float64 `json:"current_price"`
	CurrentValue       float64 `json:"current_value"`
	TotalSize          float64 `json:"total_size"` // 生命周期累计买入量，对应 v1 的 totalBought，不是当前持仓
	RealizedPnl        float64 `json:"realized_pnl"`
	UnrealizedPnl      float64 `json:"unrealized_pnl"`
	TotalPnl           float64 `json:"total_pnl"`
	PercentPnl         float64 `json:"percent_pnl"`
	PercentRealizedPnl float64 `json:"percent_realized_pnl"`
	Status             string  `json:"status"`
	Redeemable         bool    `json:"redeemable"`
	Mergeable          bool    `json:"mergeable"`
	NegativeRisk       bool    `json:"negative_risk"`
	Archived           bool    `json:"archived"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Icon               string  `json:"icon"`
	EventId            string  `json:"event_id"`
	EventSlug          string  `json:"event_slug"`
	Outcome            string  `json:"outcome"`
	// OutcomeIndex 持仓结果在市场中的下标；999 表示服务端无法判定其归属
	OutcomeIndex    int64  `json:"outcome_index"`
	OppositeOutcome string `json:"opposite_outcome"`
	OppositeTokenId string `json:"opposite_token_id"`
	EndDate         string `json:"end_date"`
	LastEventAt     int64  `json:"last_event_at"`
	FirstEntryAt    int64  `json:"first_entry_at"`
	Name            string `json:"name"`
	ProfileImage    string `json:"profile_image"`
	Verified        bool   `json:"verified"`
}
