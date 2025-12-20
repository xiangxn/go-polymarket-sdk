package polymarket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/polymarket/go-order-utils/pkg/model"
)

func GetOrderRawAmounts(side model.Side, size float64, price float64, roundConfig *RoundConfig) (model.Side, float64, float64) {
	rawPrice := RoundNormal(price, roundConfig.Price)

	if side == model.BUY {
		// force 2 decimals places
		rawTakerAmt := RoundDown(size, roundConfig.Size)

		rawMakerAmt := rawTakerAmt * rawPrice
		if DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
			rawMakerAmt = RoundUp(rawMakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
				rawMakerAmt = RoundDown(rawMakerAmt, roundConfig.Amount)
			}
		}

		return model.BUY, rawMakerAmt, rawTakerAmt

	} else {
		rawMakerAmt := RoundDown(size, roundConfig.Size)

		rawTakerAmt := rawMakerAmt * rawPrice
		if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}

		return model.SELL, rawMakerAmt, rawTakerAmt
	}
}

func GetMarketOrderRawAmounts(side model.Side, amount float64, price float64, roundConfig *RoundConfig) (model.Side, float64, float64) {
	// force 2 decimals places
	rawPrice := RoundDown(price, roundConfig.Price)

	if side == model.BUY {
		rawMakerAmt := RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt / rawPrice
		if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.BUY, rawMakerAmt, rawTakerAmt
	} else {
		rawMakerAmt := RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt * rawPrice
		if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.SELL, rawMakerAmt, rawTakerAmt
	}
}

func BuildOrderCreationArgs(signer string, maker string, signatureType model.SignatureType, userOrder *UserOrder, roundConfig *RoundConfig) (*model.OrderData, error) {
	side, rawMakerAmt, rawTakerAmt := GetOrderRawAmounts(userOrder.Side, userOrder.Size, userOrder.Price, roundConfig)
	makerAmount, err := ParseUnits(FloatToString(rawMakerAmt, 0), CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	takerAmount, err := ParseUnits(FloatToString(rawTakerAmt, 0), CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	var taker string
	if userOrder.Taker != nil {
		taker = *userOrder.Taker
	} else {
		taker = "0x0000000000000000000000000000000000000000"
	}
	var feeRateBps string
	if userOrder.FeeRateBps != nil {
		feeRateBps = FloatToString(*userOrder.FeeRateBps, 0)
	} else {
		feeRateBps = "0"
	}
	var nonce string
	if userOrder.Nonce != nil {
		nonce = strconv.FormatUint(*userOrder.Nonce, 10)
	} else {
		nonce = "0"
	}
	var expiration string
	if userOrder.Expiration != nil {
		expiration = strconv.FormatUint(*userOrder.Expiration, 10)
	} else {
		expiration = "0"
	}
	return &model.OrderData{
		Signer:        signer,
		Maker:         maker,
		Taker:         taker,
		SignatureType: signatureType,
		TokenId:       userOrder.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    feeRateBps,
		Expiration:    expiration,
		Nonce:         nonce,
	}, nil
}

func BuildMarketOrderCreationArgs(signer string, maker string, signatureType model.SignatureType, userMarketOrder *UserMarketOrder, roundConfig *RoundConfig) (*model.OrderData, error) {
	var inputPrice float64
	if userMarketOrder.Price != nil {
		inputPrice = *userMarketOrder.Price
	} else {
		inputPrice = 1
	}
	side, rawMakerAmt, rawTakerAmt := GetMarketOrderRawAmounts(userMarketOrder.Side, userMarketOrder.Amount, inputPrice, roundConfig)
	makerAmount, err := ParseUnits(FloatToString(rawMakerAmt, 0), CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	takerAmount, err := ParseUnits(FloatToString(rawTakerAmt, 0), CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	var taker string
	if userMarketOrder.Taker != nil {
		taker = *userMarketOrder.Taker
	} else {
		taker = "0x0000000000000000000000000000000000000000"
	}
	var feeRateBps string
	if userMarketOrder.FeeRateBps != nil {
		feeRateBps = FloatToString(*userMarketOrder.FeeRateBps, 0)
	} else {
		feeRateBps = "0"
	}
	var nonce string
	if userMarketOrder.Nonce != nil {
		nonce = strconv.FormatUint(*userMarketOrder.Nonce, 10)
	} else {
		nonce = "0"
	}
	return &model.OrderData{
		Signer:        signer,
		Maker:         maker,
		Taker:         taker,
		SignatureType: signatureType,
		TokenId:       userMarketOrder.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    feeRateBps,
		Expiration:    "0",
		Nonce:         nonce,
	}, nil
}

func OrderToDTO(order *model.SignedOrder, owner string, orderType OrderType, deferExec bool) PostOrderDTO {
	side := BUY
	if order.Side.Int64() == int64(model.SELL) {
		side = SELL
	} else {
		side = BUY
	}
	return PostOrderDTO{
		DeferExec: deferExec,
		Order: OrderDTO{
			Salt:          order.Salt.Int64(),
			Maker:         order.Maker.String(),
			Signer:        order.Signer.String(),
			Taker:         order.Taker.String(),
			TokenId:       order.TokenId.String(),
			MakerAmount:   order.MakerAmount.String(),
			TakerAmount:   order.TakerAmount.String(),
			Expiration:    order.Expiration.String(),
			Nonce:         order.Nonce.String(),
			FeeRateBps:    order.FeeRateBps.String(),
			Side:          side,
			SignatureType: model.SignatureType(order.SignatureType.Int64()),
			Signature:     "0x" + hex.EncodeToString(order.Signature),
		},
		Owner:     owner,
		OrderType: orderType,
	}
}

// BuildHmacSignature builds an HMAC signature
// secret: base64 encoded secret key
// timestamp: Unix timestamp
// method: HTTP method (e.g., "POST", "GET")
// requestPath: API endpoint path
// body: Optional request body
// Returns: URL-safe base64 encoded HMAC signature
func GenSignature(
	secret string,
	timestamp int64,
	method string,
	requestPath string,
	body *string,
) string {
	// Build the message: timestamp + method + path + body (if present)
	message := strconv.FormatInt(timestamp, 10) + method + requestPath
	if body != nil {
		message += *body
	}
	log.Println("message: ", message)
	// Decode the base64 secret
	base64Secret, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		// If decoding fails, treat secret as raw bytes
		base64Secret = []byte(secret)
	}
	log.Println("base64Secret: ", base64Secret)

	// Create HMAC-SHA256
	h := hmac.New(sha256.New, base64Secret)
	h.Write([]byte(message))
	sig := h.Sum(nil)

	// Encode to base64
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	// Convert to URL-safe base64: '+' -> '-', '/' -> '_'
	// Keep '=' padding as is
	sigURLSafe := strings.ReplaceAll(sigBase64, "+", "-")
	sigURLSafe = strings.ReplaceAll(sigURLSafe, "/", "_")

	return sigURLSafe
}

func CreateL2Headers(signer string, creds *ApiKeyCreds, l2HeaderArgs L2HeaderArgs, timestamp *int64) map[string]string {
	if timestamp == nil {
		now := time.Now().Unix()
		timestamp = &now
	}
	signature := GenSignature(creds.Secret, *timestamp, l2HeaderArgs.Method, l2HeaderArgs.RequestPath, l2HeaderArgs.Body)
	return map[string]string{
		"POLY_ADDRESS":    signer,
		"POLY_SIGNATURE":  signature,
		"POLY_TIMESTAMP":  strconv.FormatInt(*timestamp, 10),
		"POLY_API_KEY":    creds.Key,
		"POLY_PASSPHRASE": creds.Passphrase,
	}
}

func CalculateBuyMarketPrice(books []Book, amountToMatch float64, orderType MarketOrderType) (float64, error) {
	length := len(books)
	if length == 0 {
		return 0, fmt.Errorf("no match")
	}
	sum := 0.0
	/*
	   Asks:
	   [
	       { price: '0.6', size: '100' },
	       { price: '0.55', size: '100' },
	       { price: '0.5', size: '100' }
	   ]
	   So, if the amount to match is $150 that will be reached at first position so price will be 0.6
	*/
	for i := length - 1; i >= 0; i-- {
		p := books[i]
		sum += p.Size * p.Price
		if sum >= amountToMatch {
			return p.Price, nil
		}
	}
	if orderType == MARKET_FOK {
		return 0, fmt.Errorf("no match")
	}
	return books[0].Price, nil
}

func CalculateSellMarketPrice(books []Book, amountToMatch float64, orderType MarketOrderType) (float64, error) {
	length := len(books)
	if length == 0 {
		return 0, fmt.Errorf("no match")
	}
	sum := 0.0
	/*
	   Bids:
	   [
	       { price: '0.4', size: '100' },
	       { price: '0.45', size: '100' },
	       { price: '0.5', size: '100' }
	   ]
	   So, if the amount to match is 300 that will be reached at the first position so price will be 0.4
	*/
	for i := length - 1; i >= 0; i-- {
		p := books[i]
		sum += p.Size
		if sum >= amountToMatch {
			return p.Price, nil
		}
	}
	if orderType == MARKET_FOK {
		return 0, fmt.Errorf("no match")
	}
	return books[0].Price, nil
}
