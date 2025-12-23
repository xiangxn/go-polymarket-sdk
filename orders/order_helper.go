package orders

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/xiangxn/go-polymarket-sdk/constants"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func GetOrderRawAmounts(side model.Side, size float64, price float64, roundConfig *RoundConfig) (model.Side, float64, float64) {
	rawPrice := utils.RoundNormal(price, roundConfig.Price)

	if side == model.BUY {
		// force 2 decimals places
		rawTakerAmt := utils.RoundDown(size, roundConfig.Size)

		rawMakerAmt := rawTakerAmt * rawPrice
		if utils.DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
			rawMakerAmt = utils.RoundUp(rawMakerAmt, roundConfig.Amount+4)
			if utils.DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
				rawMakerAmt = utils.RoundDown(rawMakerAmt, roundConfig.Amount)
			}
		}

		return model.BUY, rawMakerAmt, rawTakerAmt

	} else {
		rawMakerAmt := utils.RoundDown(size, roundConfig.Size)

		rawTakerAmt := rawMakerAmt * rawPrice
		if utils.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = utils.RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if utils.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = utils.RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}

		return model.SELL, rawMakerAmt, rawTakerAmt
	}
}

func GetMarketOrderRawAmounts(side model.Side, amount float64, price float64, roundConfig *RoundConfig) (model.Side, float64, float64) {
	// force 2 decimals places
	rawPrice := utils.RoundDown(price, roundConfig.Price)

	if side == model.BUY {
		rawMakerAmt := utils.RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt / rawPrice
		if utils.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = utils.RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if utils.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = utils.RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.BUY, rawMakerAmt, rawTakerAmt
	} else {
		rawMakerAmt := utils.RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt * rawPrice
		if utils.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = utils.RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if utils.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = utils.RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}
		return model.SELL, rawMakerAmt, rawTakerAmt
	}
}

func BuildOrderCreationArgs(signer string, maker string, signatureType model.SignatureType, userOrder *UserOrder, roundConfig *RoundConfig) (*model.OrderData, error) {
	side, rawMakerAmt, rawTakerAmt := GetOrderRawAmounts(userOrder.Side, userOrder.Size, userOrder.Price, roundConfig)
	makerAmount, err := utils.ParseUnits(utils.FloatToString(rawMakerAmt, 0), constants.CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	takerAmount, err := utils.ParseUnits(utils.FloatToString(rawTakerAmt, 0), constants.CollateralTokenDecimals)
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
		feeRateBps = utils.FloatToString(*userOrder.FeeRateBps, 0)
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
	makerAmount, err := utils.ParseUnits(utils.FloatToString(rawMakerAmt, 0), constants.CollateralTokenDecimals)
	if err != nil {
		return nil, err
	}
	takerAmount, err := utils.ParseUnits(utils.FloatToString(rawTakerAmt, 0), constants.CollateralTokenDecimals)
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
		feeRateBps = utils.FloatToString(*userMarketOrder.FeeRateBps, 0)
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
