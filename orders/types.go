package orders

import (
	"github.com/polymarket/go-order-utils/pkg/model"
)

type MarketOrderType string

const (
	MARKET_FOK MarketOrderType = "FOK"
	MARKET_FAK MarketOrderType = "FAK"
)

type TickSize string

const (
	TickSize01    TickSize = "0.1"
	TickSize001   TickSize = "0.01"
	TickSize0001  TickSize = "0.001"
	TickSize00001 TickSize = "0.0001"
)

type OrderType string

const (
	GTC OrderType = "GTC"
	FOK OrderType = "FOK"
	GTD OrderType = "GTD"
	FAK OrderType = "FAK"
)

type SideType string

const (
	BUY  SideType = "BUY"
	SELL SideType = "SELL"
)

type PostOrderDTO struct {
	DeferExec bool      `json:"deferExec"`
	Order     OrderDTO  `json:"order"`
	Owner     string    `json:"owner"`
	OrderType OrderType `json:"orderType"`
}

type PostOrdersArgs struct {
	Order     *model.SignedOrder
	OrderType OrderType
}

type OrderDTO struct {
	Salt int64 `json:"salt"`
	// 订单的发起人，即订单的资金来源。
	Maker string `json:"maker"`

	// 订单签署人。此项为可选，若未填写，则签署人即为订单制作人。
	Signer string `json:"signer"`

	// 下单的地址。地址零表示公共订单。
	Taker string `json:"taker"`

	// 要买卖的 CTF ERC1155 资产的 Token ID。
	// 如果是买入，则为要购买的资产的 Token ID，即 makerAssetId。
	// 如果是卖出，则为要出售的资产的 Token ID，即 takerAssetId。
	TokenId string `json:"tokenId"`

	// Maker 数量，即要出售的代币最大数量。
	MakerAmount string `json:"makerAmount"`

	// Taker 数量，即接收的最小代币数量
	TakerAmount string `json:"takerAmount"`

	// 订单方向，买入或卖出, BUY or SELL
	Side SideType `json:"side"`

	// 订单过期的时间戳。
	// 可选，如果未指定，则值为“0”（无过期时间）。
	Expiration string `json:"expiration"`

	// 用于链上取消的随机数
	Nonce string `json:"nonce"`

	// 手续费率（以基点计），向委托人收取，按交易额计算
	FeeRateBps string `json:"feeRateBps"`

	// 订单使用的签名类型。默认值为“EOA”。
	SignatureType model.SignatureType `json:"signatureType"`

	Signature string `json:"signature"`
}

type RoundConfig struct {
	Price  int
	Size   int
	Amount int
}

type CreateOrderOptions struct {
	TickSize      *TickSize
	SignatureType *model.SignatureType
	NegRisk       *bool
}

// UserOrder Simplified order for users
type UserOrder struct {
	/**
	 * TokenID of the Conditional token asset being traded
	 */
	TokenID string

	/**
	 * Price used to create the order
	 */
	Price float64

	/**
	 * Size in terms of the ConditionalToken
	 */
	Size float64

	/**
	 * Side of the order
	 */
	Side model.Side

	/**
	 * Fee rate, in basis points, charged to the order maker, charged on proceeds
	 */
	FeeRateBps *float64

	/**
	 * Nonce used for onchain cancellations
	 */
	Nonce *uint64

	/**
	 * Timestamp after which the order is expired.
	 */
	Expiration *uint64

	/**
	 * Address of the order taker. The zero address is used to indicate a public order
	 */
	Taker *string
}

type UserMarketOrder struct {
	/**
	 * TokenID of the Conditional token asset being traded
	 */
	TokenID string

	/**
	 * BUY orders: $$$ Amount to buy
	 * SELL orders: Shares to sell
	 */
	Amount float64

	/**
	 * Side of the order
	 */
	Side model.Side

	/**
	 * Price used to create the order
	 * If it is not present the market price will be used.
	 */
	Price *float64

	/**
	 * Fee rate, in basis points, charged to the order maker, charged on proceeds
	 */
	FeeRateBps *float64

	/**
	 * Nonce used for onchain cancellations
	 */
	Nonce *uint64

	/**
	 * Address of the order taker. The zero address is used to indicate a public order
	 */
	Taker *string

	/**
	 * Specifies the type of order execution:
	 * - FOK (Fill or Kill): The order must be filled entirely or not at all.
	 * - FAK (Fill and Kill): The order can be partially filled, and any unfilled portion is canceled.
	 */
	OrderType MarketOrderType
}

type OrderPayload struct {
	OrderID string `json:"orderID"`
}

type OpenOrderParams struct {
	Id      *string `json:"id"`
	Market  *string `json:"market"`
	AssetId *string `json:"asset_id"`
}

type OrderMarketCancelParams struct {
	Market  string `json:"market"`
	AssetId string `json:"asset_id"`
}

type OpenOrder struct {
	Id              string   `json:"id"`
	Status          string   `json:"status"`
	Owner           string   `json:"owner"`
	MakerAddress    string   `json:"maker_address"`
	Market          string   `json:"market"`
	AssetId         string   `json:"asset_id"`
	Side            string   `json:"side"`
	OriginalSize    float64  `json:"original_size"`
	SizeMatched     string   `json:"size_matched"`
	Price           float64  `json:"price"`
	AssociateTrades []string `json:"associate_trades"`
	Outcome         string   `json:"outcome"`
	CreatedAt       int64    `json:"created_at"`
	Expiration      string   `json:"expiration"`
	OrderType       string   `json:"order_type"`
}

// Book 表示价格和数量
type Book struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}
