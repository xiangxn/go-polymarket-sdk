package polymarket

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/polymarket/go-order-utils/pkg/builder"
	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/tidwall/gjson"
	Headers "github.com/xiangxn/go-polymarket-sdk/headers"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
	"resty.dev/v3"
)

type PolymarketClient struct {
	http      *resty.Client
	cfg       *Config
	signer    Signer
	tickSizes map[string]orders.TickSize
	feeRates  map[string]float64
	negRisk   map[string]bool

	muTickSizes sync.RWMutex
	muFeeRates  sync.RWMutex
	muNegRisk   sync.RWMutex
}

func NewClient(signerKey string, cfg *Config) *PolymarketClient {

	client := resty.New().SetTLSClientConfig(&tls.Config{
		MinVersion: tls.VersionTLS12,
		// 允许 session resumption
		ClientSessionCache: tls.NewLRUClientSessionCache(128),
	}).SetTransport(&http.Transport{ // 打开 KeepAlive / 连接池
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     120 * time.Second,
		ForceAttemptHTTP2:   true,
	}).SetTimeout(3 * time.Second) // 默认超时

	if cfg.SocksProxy != "" {
		client.SetProxy(cfg.SocksProxy)
	}
	if cfg.HttpTimeout > 0 {
		client.SetTimeout(cfg.HttpTimeout)
	}
	privateKey, err := crypto.HexToECDSA(signerKey)
	if err != nil {
		panic(err)
	}

	return &PolymarketClient{
		http:      client,
		cfg:       cfg,
		signer:    Signer{privateKey, crypto.PubkeyToAddress(privateKey.PublicKey)},
		tickSizes: make(map[string]orders.TickSize),
		feeRates:  make(map[string]float64),
		negRisk:   make(map[string]bool),
	}
}

func (c *PolymarketClient) ClearTickSizes() {
	c.muTickSizes.Lock()
	defer c.muTickSizes.Unlock()
	c.tickSizes = make(map[string]orders.TickSize)
}

func (c *PolymarketClient) ClearFeeRates() {
	c.muFeeRates.Lock()
	defer c.muFeeRates.Unlock()
	c.feeRates = make(map[string]float64)
}

func (c *PolymarketClient) ClearNegRisk() {
	c.muNegRisk.Lock()
	defer c.muNegRisk.Unlock()
	c.negRisk = make(map[string]bool)
}

func (c *PolymarketClient) Get(url string, params map[string]string, headers map[string]string) (*gjson.Result, error) {
	request := c.http.R()
	if params != nil {
		request.SetQueryParams(params)
	}
	Headers.OverloadHeaders(resty.MethodGet, headers)
	request.SetHeaders(headers)
	// request.SetDebug(true)
	resp, err := request.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	result := gjson.ParseBytes(resp.Bytes())
	return &result, nil
}

func (c *PolymarketClient) Post(url string, body any, headers map[string]string) (*gjson.Result, error) {
	request := c.http.R()
	if body != nil {
		request.SetBody(body)
	}
	Headers.OverloadHeaders(resty.MethodPost, headers)
	request.SetHeaders(headers)
	// request.SetDebug(true)
	resp, err := request.Post(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	result := gjson.ParseBytes(resp.Bytes())
	return &result, nil
}

func (c *PolymarketClient) Del(url string, params map[string]string, body any, headers map[string]string) (*gjson.Result, error) {

	var reqBody io.Reader
	if body != nil {
		bodyIsString := false
		if _, ok := body.(string); ok {
			bodyIsString = true
		}
		if bodyIsString {
			reqBody = bytes.NewReader([]byte(body.(string)))
		} else {
			data, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			reqBody = bytes.NewReader(data)
		}
	}

	// 构建请求
	req, err := http.NewRequest(http.MethodDelete, url, reqBody)
	if err != nil {
		return nil, err
	}

	// query params
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	// headers（先 overload 再 set）
	Headers.OverloadHeaders(http.MethodDelete, headers)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 复用 resty 的底层 http.Client（非常关键）
	client := c.http.Client()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	result := gjson.ParseBytes(respBytes)
	return &result, nil
}

func (c *PolymarketClient) FetchMarketBySlug(slug string) (*gjson.Result, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}

	url := fmt.Sprintf(
		"%s/markets/slug/%s?include_tag=true",
		c.cfg.Polymarket.GammaBaseURL,
		slug,
	)

	return c.Get(url, nil, nil)
}

func (c *PolymarketClient) GetTickSize(tokenID string) (orders.TickSize, error) {
	if tokenID == "" {
		return "", fmt.Errorf("tokenID cannot be empty")
	}
	c.muTickSizes.RLock()
	v, ok := c.tickSizes[tokenID]
	c.muTickSizes.RUnlock()
	if ok {
		return v, nil
	}

	url := fmt.Sprintf("%s/tick-size", c.cfg.Polymarket.ClobBaseURL)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return "", err
	}

	v, err = orders.NewTickSize(result.Get("minimum_tick_size").String())
	if err != nil {
		return "", err
	}
	c.muTickSizes.Lock()
	c.tickSizes[tokenID] = v
	c.muTickSizes.Unlock()

	return v, nil
}

func (c *PolymarketClient) GetFeeRateBps(tokenID string) (float64, error) {
	if tokenID == "" {
		return 0, fmt.Errorf("tokenID cannot be empty")
	}
	c.muFeeRates.RLock()
	v, ok := c.feeRates[tokenID]
	c.muFeeRates.RUnlock()
	if ok {
		return v, nil
	}
	url := fmt.Sprintf("%s/fee-rate", c.cfg.Polymarket.ClobBaseURL)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return 0, err
	}
	c.muFeeRates.Lock()
	c.feeRates[tokenID] = result.Get("base_fee").Float()
	c.muFeeRates.Unlock()
	return c.feeRates[tokenID], nil
}

func (c *PolymarketClient) GetNegRisk(tokenID string) (bool, error) {
	if tokenID == "" {
		return false, fmt.Errorf("tokenID cannot be empty")
	}
	c.muNegRisk.RLock()
	v, ok := c.negRisk[tokenID]
	c.muNegRisk.RUnlock()
	if ok {
		return v, nil
	}
	url := fmt.Sprintf("%s/neg-risk", c.cfg.Polymarket.ClobBaseURL)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return false, err
	}

	c.muNegRisk.Lock()
	c.negRisk[tokenID] = result.Get("neg_risk").Bool()
	c.muNegRisk.Unlock()
	return c.negRisk[tokenID], nil
}

func (c *PolymarketClient) ResolveTickSize(tokenID string, tickSize *orders.TickSize) (orders.TickSize, error) {
	minTickSize, err := c.GetTickSize(tokenID)
	if err != nil {
		return "", err
	}
	if tickSize != nil {
		if orders.IsTickSizeSmaller(*tickSize, minTickSize) {
			return "", fmt.Errorf("tickSize %s is smaller than minTickSize %s", *tickSize, minTickSize)
		}
		return *tickSize, nil
	}
	return minTickSize, nil
}

func (c *PolymarketClient) ResolveFeeRateBps(tokenID string, userFeeRateBps *float64) (float64, error) {
	marketFeeRateBps, err := c.GetFeeRateBps(tokenID)
	if err != nil {
		return 0, err
	}
	if marketFeeRateBps > 0 && userFeeRateBps != nil && userFeeRateBps != &marketFeeRateBps {
		return 0, fmt.Errorf("userFeeRateBps %f is not equal to marketFeeRateBps %f", *userFeeRateBps, marketFeeRateBps)
	}
	return marketFeeRateBps, nil
}

func (c *PolymarketClient) CreateOrder(userOrder *orders.UserOrder, options orders.CreateOrderOptions) (*model.SignedOrder, error) {
	if c.cfg.Polymarket.ChainID == nil {
		return nil, fmt.Errorf("chainID cannot be empty")
	}
	tickSize, err := c.ResolveTickSize(userOrder.TokenID, options.TickSize)
	if err != nil {
		return nil, err
	}
	feeRateBps, err := c.ResolveFeeRateBps(userOrder.TokenID, userOrder.FeeRateBps)
	if err != nil {
		return nil, err
	}
	userOrder.FeeRateBps = &feeRateBps

	if !orders.PriceValid(userOrder.Price, tickSize) {
		return nil, fmt.Errorf("invalid price (%.4f), min: %s - max: %.4f", userOrder.Price, tickSize, 1-orders.ConvertTickSize(tickSize))
	}
	var negRisk bool
	if options.NegRisk != nil {
		negRisk = *options.NegRisk
	} else {
		negRisk, err = c.GetNegRisk(userOrder.TokenID)
		if err != nil {
			return nil, err
		}
	}

	var nr model.VerifyingContract
	if negRisk {
		nr = model.NegRiskCTFExchange
	} else {
		nr = model.CTFExchange
	}
	builder := builder.NewExchangeOrderBuilderImpl(c.cfg.Polymarket.ChainID, nil)

	var maker string
	if c.cfg.Polymarket.FunderAddress != nil {
		maker = *c.cfg.Polymarket.FunderAddress
	} else {
		maker = c.signer.Address.String()
	}
	signatureType := model.EOA
	if options.SignatureType != nil {
		signatureType = *options.SignatureType
	}
	orderData, err := orders.BuildOrderCreationArgs(c.signer.Address.String(), maker, signatureType, userOrder, orders.GetRoundConfig(tickSize))
	if err != nil {
		return nil, err
	}
	order, err := builder.BuildSignedOrder(c.signer.PrivateKey, orderData, nr)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (c *PolymarketClient) CreateMarketOrder(userMarketOrder *orders.UserMarketOrder, options orders.CreateOrderOptions) (*model.SignedOrder, error) {
	if c.cfg.Polymarket.ChainID == nil {
		return nil, fmt.Errorf("chainID cannot be empty")
	}
	tickSize, err := c.ResolveTickSize(userMarketOrder.TokenID, options.TickSize) // 建议市场开始时就获取tickSize
	if err != nil {
		return nil, err
	}
	feeRateBps, err := c.ResolveFeeRateBps(userMarketOrder.TokenID, userMarketOrder.FeeRateBps) // 建议市场开始时就获取feeRateBps
	if err != nil {
		return nil, err
	}
	userMarketOrder.FeeRateBps = &feeRateBps

	if userMarketOrder.Price == nil { // 尽量在外面计算价格
		price, cErr := c.CalculateMarketPrice(userMarketOrder.TokenID, userMarketOrder.Side, userMarketOrder.Amount, userMarketOrder.OrderType)
		if cErr != nil {
			return nil, cErr
		}
		userMarketOrder.Price = &price
	}

	if !orders.PriceValid(*userMarketOrder.Price, tickSize) {
		return nil, fmt.Errorf("invalid price (%.4f), min: %s - max: %.4f", *userMarketOrder.Price, tickSize, 1-orders.ConvertTickSize(tickSize))
	}

	var negRisk bool
	if options.NegRisk != nil {
		negRisk = *options.NegRisk
	} else {
		negRisk, err = c.GetNegRisk(userMarketOrder.TokenID)
		if err != nil {
			return nil, err
		}
	}
	var nr model.VerifyingContract
	if negRisk {
		nr = model.NegRiskCTFExchange
	} else {
		nr = model.CTFExchange
	}

	builder := builder.NewExchangeOrderBuilderImpl(c.cfg.Polymarket.ChainID, nil)
	var maker string
	if c.cfg.Polymarket.FunderAddress != nil {
		maker = *c.cfg.Polymarket.FunderAddress
	} else {
		maker = c.signer.Address.String()
	}
	signatureType := model.EOA
	if options.SignatureType != nil {
		signatureType = *options.SignatureType
	}
	orderData, err := orders.BuildMarketOrderCreationArgs(c.signer.Address.String(), maker, signatureType, userMarketOrder, orders.GetRoundConfig(tickSize))
	if err != nil {
		return nil, err
	}
	order, err := builder.BuildSignedOrder(c.signer.PrivateKey, orderData, nr)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (c *PolymarketClient) PostOrder(order *model.SignedOrder, orderType orders.OrderType, deferExec bool) (*gjson.Result, error) {
	path := "/order"
	if c.cfg.Polymarket.HasCLOBAuth() == false {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)
	orderPayload := orders.OrderToDTO(order, c.cfg.Polymarket.CLOBCreds.Key, orderType, deferExec)

	data, err := json.Marshal(orderPayload)
	if err != nil {
		return nil, err
	}
	body := string(data)
	log.Printf("body: %s", body)
	l2HeaderArgs := Headers.L2HeaderArgs{
		Method:      "POST",
		RequestPath: path,
		Body:        &body,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &l2HeaderArgs, nil)

	if c.cfg.Polymarket.HasBuilderAuth() {
		builderHeaders := Headers.CreateBuilderHeaders(c.cfg.Polymarket.BuilderCreds, resty.MethodPost, path, &body, nil)

		if builderHeaders != nil {
			maps.Copy(headers, builderHeaders)
		}
	}
	return c.Post(url, body, headers)
}

func (c *PolymarketClient) CancelOrder(payload *orders.OrderPayload) (*gjson.Result, error) {
	path := "/order"
	if c.cfg.Polymarket.HasCLOBAuth() == false {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)
	data, err := json.Marshal(*payload)
	if err != nil {
		return nil, err
	}
	body := string(data)
	log.Printf("body: %s", body)

	l2HeaderArgs := Headers.L2HeaderArgs{
		Method:      "DELETE",
		RequestPath: path,
		Body:        &body,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &l2HeaderArgs, nil)
	return c.Del(url, nil, body, headers)
}

func (c *PolymarketClient) PostOrders(args []orders.PostOrdersArgs, deferExec bool) (*gjson.Result, error) {
	path := "/orders"
	if c.cfg.Polymarket.HasCLOBAuth() == false {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)

	var ordersPayload []orders.PostOrderDTO
	for _, arg := range args {
		orderPayload := orders.OrderToDTO(arg.Order, c.cfg.Polymarket.CLOBCreds.Key, arg.OrderType, deferExec)
		ordersPayload = append(ordersPayload, orderPayload)
	}

	data, err := json.Marshal(ordersPayload)
	if err != nil {
		return nil, err
	}
	body := string(data)

	l2HeaderArgs := Headers.L2HeaderArgs{
		Method:      "POST",
		RequestPath: path,
		Body:        &body,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &l2HeaderArgs, nil)

	if c.cfg.Polymarket.HasBuilderAuth() {
		builderHeaders := Headers.CreateBuilderHeaders(c.cfg.Polymarket.BuilderCreds, resty.MethodPost, path, &body, nil)

		if builderHeaders != nil {
			maps.Copy(headers, builderHeaders)
		}
	}
	return c.Post(url, body, headers)
}

func (c *PolymarketClient) CancelOrders(ordersHashes []string) (*gjson.Result, error) {
	if c.cfg.Polymarket.HasCLOBAuth() == false {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	path := "/orders"
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)

	data, err := json.Marshal(ordersHashes)
	if err != nil {
		return nil, err
	}
	body := string(data)

	l2HeaderArgs := Headers.L2HeaderArgs{
		Method:      "DELETE",
		RequestPath: path,
		Body:        &body,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &l2HeaderArgs, nil)
	return c.Del(url, nil, body, headers)
}

func (c *PolymarketClient) GetOpenOrders(params *orders.OpenOrderParams, onlyFirstPage bool, nextCursor *string) ([]orders.OpenOrder, error) {
	if c.cfg.Polymarket.HasCLOBAuth() == false {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	path := "/data/orders"
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)

	l2HeaderArgs := Headers.L2HeaderArgs{
		Method:      "GET",
		RequestPath: path,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &l2HeaderArgs, nil)

	var openOrders []orders.OpenOrder
	if nextCursor == nil {
		nextCursor = utils.StringPtr("MA==")
	}
	for *nextCursor != "LTE=" && (*nextCursor == "MA==" || !onlyFirstPage) {
		pms := map[string]string{
			"next_cursor": *nextCursor,
		}
		if params != nil {
			if params.Market != nil {
				pms["market"] = *params.Market
			}
			if params.Id != nil {
				pms["id"] = *params.Id
			}
			if params.AssetId != nil {
				pms["asset_id"] = *params.AssetId
			}
		}
		result, err := c.Get(url, pms, headers)
		if err != nil {
			return nil, err
		}
		nextCursor = utils.StringPtr(result.Get("next_cursor").String())
		for _, item := range result.Get("data").Array() {
			openOrders = append(openOrders, orders.OpenOrder{
				Id:              item.Get("id").String(),
				Status:          item.Get("status").String(),
				Owner:           item.Get("owner").String(),
				MakerAddress:    item.Get("maker_address").String(),
				Market:          item.Get("market").String(),
				AssetId:         item.Get("asset_id").String(),
				Side:            item.Get("side").String(),
				OriginalSize:    item.Get("original_size").Float(),
				SizeMatched:     item.Get("size_matched").String(),
				Price:           item.Get("price").Float(),
				AssociateTrades: utils.GetStringArray(&item, "associate_trades"),
				Outcome:         item.Get("outcome").String(),
				CreatedAt:       item.Get("created_at").Int(),
				Expiration:      item.Get("expiration").String(),
				OrderType:       item.Get("order_type").String(),
			})
		}
	}
	return openOrders, nil
}

func (c *PolymarketClient) GetApiKeys() ([]string, error) {
	if c.cfg.Polymarket.HasCLOBAuth() == false {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	path := "/auth/api-keys"
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)

	headerArgs := Headers.L2HeaderArgs{
		Method:      "GET",
		RequestPath: path,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &headerArgs, nil)
	result, err := c.Get(url, nil, headers)
	if err != nil {
		return nil, err
	}
	var apiKeys []string
	for _, item := range result.Get("apiKeys").Array() {
		apiKeys = append(apiKeys, item.Value().(string))
	}
	return apiKeys, nil
}

func (c *PolymarketClient) GetOrderBook(tokenID string) (*OrderBookSummary, error) {
	url := fmt.Sprintf("%s/book", c.cfg.Polymarket.ClobBaseURL)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return nil, err
	}
	orderBookSummary := &OrderBookSummary{
		Market:       result.Get("market").String(),
		AssetId:      result.Get("asset_id").String(),
		Timestamp:    result.Get("timestamp").Int(),
		MinOrderSize: result.Get("min_order_size").String(),
		TickSize:     result.Get("tick_size").String(),
		NegRisk:      result.Get("neg_risk").Bool(),
		Hash:         result.Get("hash").String(),
	}
	bids := result.Get("bids").Array()
	for _, item := range bids {
		orderBookSummary.Bids = append(orderBookSummary.Bids, orders.Book{
			Price: item.Get("price").Float(),
			Size:  item.Get("size").Float(),
		})
	}
	asks := result.Get("asks").Array()
	for _, item := range asks {
		orderBookSummary.Asks = append(orderBookSummary.Asks, orders.Book{
			Price: item.Get("price").Float(),
			Size:  item.Get("size").Float(),
		})
	}
	return orderBookSummary, nil
}

func (c *PolymarketClient) GetOrderBooks(params []BookParams) ([]OrderBookSummary, error) {
	url := fmt.Sprintf("%s/books", c.cfg.Polymarket.ClobBaseURL)
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	body := string(data)
	result, err := c.Post(url, body, nil)
	if err != nil {
		return nil, err
	}
	var orderBooks []OrderBookSummary
	for _, item := range result.Array() {
		orderBook := OrderBookSummary{
			Market:       item.Get("market").String(),
			AssetId:      item.Get("asset_id").String(),
			Timestamp:    item.Get("timestamp").Int(),
			MinOrderSize: item.Get("min_order_size").String(),
			TickSize:     item.Get("tick_size").String(),
			NegRisk:      item.Get("neg_risk").Bool(),
			Hash:         item.Get("hash").String(),
		}
		bids := item.Get("bids").Array()
		for _, it := range bids {
			orderBook.Bids = append(orderBook.Bids, orders.Book{
				Price: it.Get("price").Float(),
				Size:  it.Get("size").Float(),
			})
		}
		asks := item.Get("asks").Array()
		for _, it := range asks {
			orderBook.Asks = append(orderBook.Asks, orders.Book{
				Price: it.Get("price").Float(),
				Size:  it.Get("size").Float(),
			})
		}
		orderBooks = append(orderBooks, orderBook)
	}
	return orderBooks, nil
}

func (c *PolymarketClient) GetServerTime() (int64, error) {
	url := fmt.Sprintf("%s/time", c.cfg.Polymarket.ClobBaseURL)
	result, err := c.Get(url, nil, nil)
	if err != nil {
		return 0, err
	}
	return result.Int(), nil
}

func (c *PolymarketClient) CalculateMarketPrice(tokenID string, side model.Side, amount float64, orderType orders.MarketOrderType) (float64, error) {
	book, err := c.GetOrderBook(tokenID)
	if err != nil {
		return 0, fmt.Errorf("no orderbook")
	}
	if side == model.BUY {
		if book.Asks == nil {
			return 0, fmt.Errorf("no match")
		}
		return orders.CalculateBuyMarketPrice(book.Asks, amount, orderType)
	} else {
		if book.Bids == nil {
			return 0, fmt.Errorf("no match")
		}
		return orders.CalculateSellMarketPrice(book.Bids, amount, orderType)
	}
}

func (c *PolymarketClient) CancelMarketOrders(payload *orders.OrderMarketCancelParams) (*gjson.Result, error) {
	path := "/cancel-market-orders"
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.ClobBaseURL, path)

	data, err := json.Marshal(*payload)
	if err != nil {
		return nil, err
	}
	body := string(data)

	l2HeaderArgs := Headers.L2HeaderArgs{
		Method:      "DELETE",
		RequestPath: path,
		Body:        &body,
	}
	headers := Headers.CreateL2Headers(c.signer.Address, c.cfg.Polymarket.CLOBCreds, &l2HeaderArgs, nil)
	return c.Del(url, nil, body, headers)
}

func (c *PolymarketClient) SearchPositions(proxyWallet string, redeemable bool) (*gjson.Result, error) {
	url := fmt.Sprintf("%s%s", c.cfg.Polymarket.DataAPIBaseURL, "/positions")
	params := map[string]string{
		"sizeThreshold": "0",
		"limit":         "100",
		"sortBy":        "TOKENS",
		"sortDirection": "DESC",
		"user":          proxyWallet,
	}
	if redeemable {
		params["redeemable"] = "true"
	}
	result, err := c.Get(url, params, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *PolymarketClient) SetTickSize(tokenID string, tickSize float64) error {
	v, err := orders.NewTickSize(utils.FloatToString(tickSize, 0))
	if err != nil {
		return err
	}

	c.muTickSizes.Lock()
	defer c.muTickSizes.Unlock()

	c.tickSizes[tokenID] = v
	return nil
}

func (c *PolymarketClient) SetFeeRateBps(tokenID string, feeRateBps float64) {
	c.muFeeRates.Lock()
	defer c.muFeeRates.Unlock()

	c.feeRates[tokenID] = feeRateBps
}

func (c *PolymarketClient) SetNegRisk(tokenID string, negRisk bool) {
	c.muNegRisk.Lock()
	defer c.muNegRisk.Unlock()

	c.negRisk[tokenID] = negRisk
}
