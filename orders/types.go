package orders

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
	POST_BUY  SideType = "BUY"
	POST_SELL SideType = "SELL"
)

type PostOrderDTO struct {
	DeferExec bool      `json:"deferExec"`
	Order     OrderDTO  `json:"order"`
	Owner     string    `json:"owner"`
	OrderType OrderType `json:"orderType"`
}

type PostOrdersArgs struct {
	Order     *SignedOrder
	OrderType OrderType
}

type OrderDTO struct {
	// Ethereum address of the maker (In the default case, this is your proxy address)
	Maker string `json:"maker"`

	// 订单签署人。此项为可选，若未填写，则签署人即为订单制作人。
	Signer string `json:"signer"`

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
	// Unix timestamp when the order expires. Present in the API wire body; not part of the CLOB V2 EIP-712 signed order struct.
	Expiration string `json:"expiration"`

	// Unix timestamp in milliseconds when the order was created (used for order uniqueness)
	Timestamp string `json:"timestamp"`

	// Builder code (bytes32) for integrator attribution. 0x + 64 hex chars or empty.
	Builder string `json:"builder"`

	Signature string `json:"signature"`

	Salt int64 `json:"salt"`

	// 订单使用的签名类型。默认值为“EOA”。
	SignatureType SignatureType `json:"signatureType"`

	Metadata string `json:"metadata"`
}

type RoundConfig struct {
	Price  int
	Size   int
	Amount int
}

type CreateOrderOptions struct {
	TickSize      *TickSize
	SignatureType *SignatureType
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
	Side Side

	/**
	 * Metadata (bytes32)
	 */
	Metadata *string

	/**
	 * Builder code (bytes32)
	 */
	BuilderCode *string

	/**
	 * Expiration timestamp (unix seconds). Defaults to 0 (no expiration).
	 */
	Expiration *string
}

type UserMarketOrder struct {
	/**
	 * TokenID of the Conditional token asset being traded
	 */
	TokenID string

	/**
	 * Price used to create the order
	 * If it is not present the market price will be used.
	 */
	Price *float64

	/**
	 * BUY orders: $$$ Amount to buy
	 * SELL orders: Shares to sell
	 */
	Amount float64

	/**
	 * Side of the order
	 */
	Side Side

	/**
	 * Specifies the type of order execution:
	 * - FOK (Fill or Kill): The order must be filled entirely or not at all.
	 * - FAK (Fill and Kill): The order can be partially filled, and any unfilled portion is canceled.
	 */
	OrderType MarketOrderType

	/**
	 * User's USDC balance. If provided and sufficient to cover amount + fees, the order
	 * amount is used as-is. Otherwise fees are deducted from the amount.
	 * If this field is left empty, the default flow is to use the order amount as-is
	 */
	UserUSDCBalance *float64

	/**
	 * Metadata (bytes32)
	 */
	Metadata *string

	/**
	 * Builder code (bytes32)
	 */
	BuilderCode *string
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
	OriginalSize    float64  `json:"original_size,string"`
	SizeMatched     float64  `json:"size_matched,string"`
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
