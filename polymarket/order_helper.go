package polymarket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/polymarket/go-order-utils/pkg/model"
)

func GetOrderRawAmounts(side model.Side, size float64, price float64, roundConfig RoundConfig) (model.Side, float64, float64) {
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

func BuildOrderCreationArgs(signer string, maker string, signatureType model.SignatureType, userOrder UserOrder, roundConfig RoundConfig) (model.OrderData, error) {
	side, rawMakerAmt, rawTakerAmt := GetOrderRawAmounts(userOrder.Side, userOrder.Size, userOrder.Price, roundConfig)
	makerAmount, err := ParseUnits(FloatToString(rawMakerAmt, 0), CollateralTokenDecimals)
	if err != nil {
		return model.OrderData{}, err
	}
	takerAmount, err := ParseUnits(FloatToString(rawTakerAmt, 0), CollateralTokenDecimals)
	if err != nil {
		return model.OrderData{}, err
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
	return model.OrderData{
		Signer:        signer,
		Maker:         maker,
		Taker:         taker,
		SignatureType: signatureType,
		TokenId:       userOrder.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    feeRateBps,
		Nonce:         nonce,
	}, nil
}

func OrderToDTO(order *model.SignedOrder, owner string, orderType OrderType, deferExec bool) PostOrderDTO {
	side := BUY
	if order.Side.Int64() == int64(model.SELL) {
		side = BUY
	} else {
		side = SELL
	}
	return PostOrderDTO{
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
			Signature:     string(order.Signature),
		},
		Owner:     owner,
		OrderType: orderType,
		DeferExec: deferExec,
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

	// Decode the base64 secret
	base64Secret, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		// If decoding fails, treat secret as raw bytes
		base64Secret = []byte(secret)
	}

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
