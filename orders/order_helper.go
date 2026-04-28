package orders

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/constants"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func GetOrderRawAmounts(side Side, size float64, price float64, roundConfig *RoundConfig) (Side, float64, float64) {
	rawPrice := utils.RoundNormal(price, roundConfig.Price)

	if side == BUY {
		// force 2 decimals places
		rawTakerAmt := utils.RoundDown(size, roundConfig.Size)

		// log.Printf("rawTakerAmt: %f, size: %f, Amount: %d", rawTakerAmt, size, roundConfig.Amount)
		rawMakerAmt := rawTakerAmt * rawPrice
		rawMakerAmt = utils.RoundDown(rawMakerAmt, roundConfig.Amount)

		// log.Printf("rawMakerAmt: %f", rawMakerAmt)
		return BUY, rawMakerAmt, rawTakerAmt

	} else {
		rawMakerAmt := utils.RoundDown(size, roundConfig.Size)

		rawTakerAmt := rawMakerAmt * rawPrice
		rawTakerAmt = utils.RoundDown(rawTakerAmt, roundConfig.Amount)

		return SELL, rawMakerAmt, rawTakerAmt
	}
}

func GetMarketOrderRawAmounts(side Side, amount float64, price float64, roundConfig *RoundConfig) (Side, float64, float64) {
	// force 2 decimals places
	rawPrice := utils.RoundDown(price, roundConfig.Price)

	if side == BUY {
		rawMakerAmt := utils.RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt / rawPrice
		rawTakerAmt = utils.RoundDown(rawTakerAmt, roundConfig.Amount)
		return BUY, rawMakerAmt, rawTakerAmt
	} else {
		rawMakerAmt := utils.RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt * rawPrice
		rawTakerAmt = utils.RoundDown(rawTakerAmt, roundConfig.Amount)
		return SELL, rawMakerAmt, rawTakerAmt
	}
}

func BuildOrderCreationArgs(signer string, maker string, signatureType SignatureType, userOrder *UserOrder, roundConfig *RoundConfig) (*OrderData, error) {
	side, rawMakerAmt, rawTakerAmt := GetOrderRawAmounts(userOrder.Side, userOrder.Size, userOrder.Price, roundConfig)
	// log.Printf("BuildOrderCreationArgs rawMakerAmt: %f, rawTakerAmt: %f", rawMakerAmt, rawTakerAmt)
	makerAmount, err := utils.ParseUnits(utils.FloatToString(rawMakerAmt, 0), constants.CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	takerAmount, err := utils.ParseUnits(utils.FloatToString(rawTakerAmt, 0), constants.CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	return &OrderData{
		Maker:         maker,
		TokenId:       userOrder.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		Signer:        &signer,
		SignatureType: &signatureType,
		Timestamp:     &timestamp,
		Metadata:      userOrder.Metadata,
		Builder:       userOrder.BuilderCode,
		Expiration:    userOrder.Expiration,
	}, nil
}

func BuildMarketOrderCreationArgs(signer string, maker string, signatureType SignatureType, userMarketOrder *UserMarketOrder, roundConfig *RoundConfig) (*OrderData, error) {
	var inputPrice float64
	if userMarketOrder.Price != nil {
		inputPrice = *userMarketOrder.Price
	} else {
		inputPrice = 1
	}
	side, rawMakerAmt, rawTakerAmt := GetMarketOrderRawAmounts(userMarketOrder.Side, userMarketOrder.Amount, inputPrice, roundConfig)
	makerAmount, err := utils.ParseUnits(utils.FloatToString(rawMakerAmt, 0), constants.CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	takerAmount, err := utils.ParseUnits(utils.FloatToString(rawTakerAmt, 0), constants.CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	expiration := "0"

	return &OrderData{
		Maker:         maker,
		TokenId:       userMarketOrder.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		Signer:        &signer,
		SignatureType: &signatureType,
		Timestamp:     &timestamp,
		Metadata:      userMarketOrder.Metadata,
		Builder:       userMarketOrder.BuilderCode,
		Expiration:    &expiration,
	}, nil
}

func OrderToDTO(order *SignedOrder, owner string, orderType OrderType, deferExec bool, expiration string) PostOrderDTO {
	side := POST_BUY
	if order.Side.Int64() == int64(SELL) {
		side = POST_SELL
	} else {
		side = POST_BUY
	}
	return PostOrderDTO{
		DeferExec: deferExec,
		Order: OrderDTO{
			Maker:         order.Maker.String(),
			Signer:        order.Signer.String(),
			TokenId:       order.TokenId.String(),
			MakerAmount:   order.MakerAmount.String(),
			TakerAmount:   order.TakerAmount.String(),
			Side:          side,
			Expiration:    expiration,
			Timestamp:     order.Timestamp.String(),
			Builder:       order.Builder.Hex(),
			Signature:     "0x" + hex.EncodeToString(order.Signature),
			Salt:          order.Salt.Int64(),
			SignatureType: SignatureType(order.SignatureType.Int64()),
			Metadata:      order.Metadata.Hex(),
		},
		Owner:     owner,
		OrderType: orderType,
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
