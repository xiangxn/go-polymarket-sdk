package main

// ----------------------
// 数据结构定义
// ----------------------

// Book 表示价格和数量
type Book struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

// PriceData 表示价格数据
type PriceData struct {
	TokenID   string `json:"tokenId"`
	BestAsk   Book   `json:"bestAsk"`
	BestBid   Book   `json:"bestBid"`
	Market    string `json:"market"`
	Timestamp int64  `json:"timestamp"`
}

// PriceUpdateCallback 价格更新回调函数类型
type PriceUpdateCallback func(priceData *PriceData)

// MarketMessage 市场订阅消息
type MarketMessage struct {
	Type      string   `json:"type"`
	AssetsIDs []string `json:"assets_ids"`
}

// BookData 订单薄数据
type BookData struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// WSMessage WebSocket消息
type WSMessage struct {
	EventType string     `json:"event_type"`
	Market    string     `json:"market"`
	AssetID   string     `json:"asset_id"`
	Bids      []BookData `json:"bids"`
	Asks      []BookData `json:"asks"`
}