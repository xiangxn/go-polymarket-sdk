package tests

import (
	"math/big"
	"os"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/builder"
	"github.com/xiangxn/go-polymarket-sdk/constants"
	"github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func TestRedeem(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.BuilderCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("BUILDER_API_KEY"),
		Secret:     os.Getenv("BUILDER_SECRET"),
		Passphrase: os.Getenv("BUILDER_PASSPHRASE"),
	}
	privateKey := os.Getenv("SIGNERKEY")
	funderAddress := os.Getenv("FUNDERADDRESS")
	client := polymarket.NewClient(privateKey, config)

	positions, err := client.SearchPositions(funderAddress, true, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("positions: %+v", positions)

	conditionIds := []string{}
	negRisks := []bool{}
	amounts := [][]*big.Int{}
	for _, position := range positions.Array() {
		conditionIds = append(conditionIds, position.Get("conditionId").String())
		negRisk := position.Get("negativeRisk").Bool()
		if negRisk {
			ams := []*big.Int{new(big.Int).SetInt64(0), new(big.Int).SetInt64(0)}
			value, _ := utils.ParseUnits(position.Get("size").String(), constants.CollateralTokenDecimals)
			ams[position.Get("outcomeIndex").Int()] = value
			amounts = append(amounts, ams)
		} else {
			amounts = append(amounts, []*big.Int{})
		}
		negRisks = append(negRisks, negRisk)
	}

	// if len(conditionIds) <= 0 {
	// 	t.Log("no positions to redeem")
	// 	return
	// }
	relayClient := builder.NewRelayClient(config.Polymarket.RelayerBaseURL, privateKey, 137, config.Polymarket.BuilderCreds, nil)
	result, err := relayClient.RedeemBatch(conditionIds, negRisks, amounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}
