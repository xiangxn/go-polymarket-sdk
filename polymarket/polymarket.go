package polymarket

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/crypto"
	builderSDK "github.com/polymarket/go-builder-signing-sdk"
	"github.com/polymarket/go-order-utils/pkg/builder"
	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

type PolymarketClient struct {
	clobHost     string
	http         *resty.Client
	cfg          Config
	signer       Signer
	creds        *ApiKeyCreds
	builderCreds *builderSDK.LocalSignerConfig
	tickSizes    map[string]TickSize
	feeRates     map[string]float64
	negRisk      map[string]bool
}

func NewClient(signerKey string, cfg Config) *PolymarketClient {

	client := resty.New()
	if cfg.SocksProxy != nil {
		client.SetProxy(*cfg.SocksProxy)
	}
	if cfg.Timeout > 0 {
		client.SetTimeout(cfg.Timeout)
	}
	privateKey, err := crypto.HexToECDSA(signerKey)
	if err != nil {
		panic(err)
	}

	return &PolymarketClient{
		http:         client,
		cfg:          cfg,
		signer:       Signer{privateKey, crypto.PubkeyToAddress(privateKey.PublicKey)},
		tickSizes:    make(map[string]TickSize),
		feeRates:     make(map[string]float64),
		negRisk:      make(map[string]bool),
		clobHost:     "https://clob.polymarket.com",
		creds:        cfg.CLOBCreds,
		builderCreds: cfg.BuilderCreds,
	}
}

func (c *PolymarketClient) Get(url string, params map[string]string, headers map[string]string) (*gjson.Result, error) {
	request := c.http.R()
	if params != nil {
		request.SetQueryParams(params)
	}
	if headers != nil {
		request.SetHeaders(headers)
	}
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
	if headers != nil {
		request.SetHeaders(headers)
	}
	if body != nil {
		request.SetBody(body)
	}
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
	request := c.http.R()
	if params != nil {
		request.SetQueryParams(params)
	}
	if headers != nil {
		request.SetHeaders(headers)
	}
	if body != nil {
		request.SetBody(body)
	}
	resp, err := request.Delete(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	result := gjson.ParseBytes(resp.Bytes())
	return &result, nil
}

func (c *PolymarketClient) FetchMarketBySlug(slug string) (*gjson.Result, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}

	url := fmt.Sprintf(
		"https://gamma-api.polymarket.com/markets/slug/%s?include_tag=true",
		slug,
	)

	return c.Get(url, nil, nil)
}

func (c *PolymarketClient) GetTickSize(tokenID string) (TickSize, error) {
	if tokenID == "" {
		return "", fmt.Errorf("tokenID cannot be empty")
	}
	v, ok := c.tickSizes[tokenID]
	if ok {
		return v, nil
	}

	url := fmt.Sprintf("%s/tick-size", c.clobHost)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return "", err
	}

	v, err = NewTickSize(result.Get("minimum_tick_size").String())
	if err != nil {
		return "", err
	}
	c.tickSizes[tokenID] = v

	return v, nil
}

func (c *PolymarketClient) GetFeeRateBps(tokenID string) (float64, error) {
	if tokenID == "" {
		return 0, fmt.Errorf("tokenID cannot be empty")
	}
	v, ok := c.feeRates[tokenID]
	if ok {
		return v, nil
	}
	url := fmt.Sprintf("%s/fee-rate", c.clobHost)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return 0, err
	}

	c.feeRates[tokenID] = result.Get("base_fee").Float()
	return c.feeRates[tokenID], nil
}

func (c *PolymarketClient) GetNegRisk(tokenID string) (bool, error) {
	if tokenID == "" {
		return false, fmt.Errorf("tokenID cannot be empty")
	}
	url := fmt.Sprintf("%s/neg-risk", c.clobHost)
	result, err := c.Get(url, map[string]string{"token_id": tokenID}, nil)
	if err != nil {
		return false, err
	}

	c.negRisk[tokenID] = result.Get("neg_risk").Bool()
	return c.negRisk[tokenID], nil
}

func (c *PolymarketClient) ResolveTickSize(tokenID string, tickSize *TickSize) (TickSize, error) {
	minTickSize, err := c.GetTickSize(tokenID)
	if err != nil {
		return "", err
	}
	if tickSize != nil {
		if IsTickSizeSmaller(*tickSize, minTickSize) {
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

func (c *PolymarketClient) CreateOrder(userOrder UserOrder, options CreateOrderOptions) (*model.SignedOrder, error) {
	if options.chainID == nil {
		return nil, fmt.Errorf("chainID cannot be empty")
	}
	tickSize, err := c.ResolveTickSize(userOrder.TokenID, &options.TickSize)
	if err != nil {
		return nil, err
	}
	feeRateBps, err := c.ResolveFeeRateBps(userOrder.TokenID, userOrder.FeeRateBps)
	if err != nil {
		return nil, err
	}
	userOrder.FeeRateBps = &feeRateBps

	if !PriceValid(userOrder.Price, tickSize) {
		return nil, fmt.Errorf("invalid price (%.4f), min: %s - max: %.4f", userOrder.Price, tickSize, 1-ConvertTickSize(tickSize))
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
	builder := builder.NewExchangeOrderBuilderImpl(options.chainID, nil)

	var maker string
	if options.FunderAddress != nil {
		maker = *options.FunderAddress
	} else {
		maker = c.signer.Address.String()
	}
	orderData, err := BuildOrderCreationArgs(c.signer.Address.String(), maker, options.SignatureType, userOrder, GetRoundConfig(options.TickSize))
	if err != nil {
		return nil, err
	}
	order, err := builder.BuildSignedOrder(c.signer.PrivateKey, &orderData, nr)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (c *PolymarketClient) PostOrder(order *model.SignedOrder, orderType OrderType, deferExec bool) (*gjson.Result, error) {
	path := "/order"
	if c.creds == nil {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	url := fmt.Sprintf("%s%s", c.clobHost, path)
	orderPayload := OrderToDTO(order, c.creds.Key, orderType, deferExec)

	data, err := json.Marshal(orderPayload)
	if err != nil {
		panic(err)
	}
	body := string(data)

	l2HeaderArgs := L2HeaderArgs{
		Method:      "POST",
		RequestPath: path,
		Body:        &body,
	}
	headers := CreateL2Headers(c.signer.Address.Hex(), c.creds, l2HeaderArgs, nil)

	if c.builderCreds != nil {
		signer, err := builderSDK.NewLocalSigner(*c.builderCreds)
		if err != nil {
			return nil, err
		}
		builderHeaders, err := signer.CreateHeaders(
			"POST",
			path,
			&body,
			nil,
		)
		if err != nil {
			return nil, err
		}
		if builderHeaders != nil {
			maps.Copy(headers, builderHeaders)
		}
	}
	return c.Post(url, orderPayload, headers)
}

func (c *PolymarketClient) CancelOrder(payload OrderPayload) (*gjson.Result, error) {
	path := "/order"
	if c.creds == nil {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	url := fmt.Sprintf("%s%s", c.clobHost, path)
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	body := string(data)

	l2HeaderArgs := L2HeaderArgs{
		Method:      "DELETE",
		RequestPath: path,
		Body:        &body,
	}
	headers := CreateL2Headers(c.signer.Address.Hex(), c.creds, l2HeaderArgs, nil)
	return c.Del(url, nil, payload, headers)
}

func (c *PolymarketClient) PostOrders(args []PostOrdersArgs, deferExec bool) (*gjson.Result, error) {
	path := "/orders"
	if c.creds == nil {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	url := fmt.Sprintf("%s%s", c.clobHost, path)

	var ordersPayload []PostOrderDTO
	for _, arg := range args {
		orderPayload := OrderToDTO(arg.Order, c.creds.Key, arg.OrderType, deferExec)
		ordersPayload = append(ordersPayload, orderPayload)
	}

	data, err := json.Marshal(ordersPayload)
	if err != nil {
		panic(err)
	}
	body := string(data)

	l2HeaderArgs := L2HeaderArgs{
		Method:      "POST",
		RequestPath: path,
		Body:        &body,
	}
	headers := CreateL2Headers(c.signer.Address.Hex(), c.creds, l2HeaderArgs, nil)

	if c.builderCreds != nil {
		signer, err := builderSDK.NewLocalSigner(*c.builderCreds)
		if err != nil {
			return nil, err
		}
		builderHeaders, err := signer.CreateHeaders(
			"POST",
			path,
			&body,
			nil,
		)
		if err != nil {
			return nil, err
		}
		if builderHeaders != nil {
			maps.Copy(headers, builderHeaders)
		}
	}
	return c.Post(url, ordersPayload, headers)
}

func (c *PolymarketClient) CancelOrders(ordersHashes []string) (*gjson.Result, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	path := "/orders"
	url := fmt.Sprintf("%s%s", c.clobHost, path)

	data, err := json.Marshal(ordersHashes)
	if err != nil {
		panic(err)
	}
	body := string(data)

	l2HeaderArgs := L2HeaderArgs{
		Method:      "DELETE",
		RequestPath: path,
		Body:        &body,
	}
	headers := CreateL2Headers(c.signer.Address.Hex(), c.creds, l2HeaderArgs, nil)
	return c.Del(url, nil, ordersHashes, headers)
}

func (c *PolymarketClient) GetOpenOrders(params *OpenOrderParams, onlyFirstPage bool, nextCursor *string) ([]OpenOrder, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("creds cannot be empty")
	}
	path := "/data/orders"
	url := fmt.Sprintf("%s%s", c.clobHost, path)

	l2HeaderArgs := L2HeaderArgs{
		Method:      "GET",
		RequestPath: path,
	}
	headers := CreateL2Headers(c.signer.Address.Hex(), c.creds, l2HeaderArgs, nil)

	var openOrders []OpenOrder
	if nextCursor == nil {
		nextCursor = StringPtr("MA==")
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
		nextCursor = StringPtr(result.Get("next_cursor").String())
		for _, item := range result.Get("data").Array() {
			openOrders = append(openOrders, OpenOrder{
				Id:              item.Get("id").String(),
				Status:          item.Get("status").String(),
				Owner:           item.Get("owner").String(),
				MakerAddress:    item.Get("maker_address").String(),
				Market:          item.Get("market").String(),
				AssetId:         item.Get("asset_id").String(),
				Side:            item.Get("side").String(),
				OriginalSize:    item.Get("original_size").String(),
				SizeMatched:     item.Get("size_matched").String(),
				Price:           item.Get("price").String(),
				AssociateTrades: GetStringArray(&item, "associate_trades"),
				Outcome:         item.Get("outcome").String(),
				CreatedAt:       item.Get("created_at").Uint(),
				Expiration:      item.Get("expiration").String(),
				OrderType:       item.Get("order_type").String(),
			})
		}
	}
	return openOrders, nil
}
