package tests

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/constants"
	"github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/relayer"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func TestRedeem(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.BuilderCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("BUILDER_API_KEY"),
		Secret:     os.Getenv("BUILDER_SECRET"),
		Passphrase: os.Getenv("BUILDER_PASSPHRASE"),
	}
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	funderAddress := os.Getenv("FUNDERADDRESS")
	client := polymarket.NewClient(config)

	positions, err := client.SearchPositions(&polymarket.PositionParams{
		User:   funderAddress,
		Status: polymarket.PositionRedeemable,
		Limit:  1000,
	}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("positions: %+v", positions)

	conditionIds := []string{}
	negRisks := []bool{}
	amounts := [][]*big.Int{}
	for _, position := range positions {
		negRisk := position.NegativeRisk
		var amount []*big.Int
		if negRisk {
			// v2 用 outcome_index=999 表示服务端无法判定结果归属，此时无法按下标组装金额
			if position.OutcomeIndex > 1 {
				t.Logf("skip %s: outcome_index=%d", position.ConditionId, position.OutcomeIndex)
				continue
			}
			ams := []*big.Int{new(big.Int).SetInt64(0), new(big.Int).SetInt64(0)}
			value, _ := utils.ParseUnits(utils.FloatToString(position.CurrentSize, 0), constants.CollateralTokenDecimals)
			ams[position.OutcomeIndex] = value
			amount = ams
		} else {
			amount = []*big.Int{}
		}
		conditionIds = append(conditionIds, position.ConditionId)
		negRisks = append(negRisks, negRisk)
		amounts = append(amounts, amount)
	}

	// if len(conditionIds) <= 0 {
	// 	t.Log("no positions to redeem")
	// 	return
	// }
	relayClient := relayer.NewRelayClient(config.Polymarket.RelayerBaseURL, config.Polymarket.OwnerKey, 137, config.Polymarket.BuilderCreds, nil, config.Polymarket.RelayerKey)
	result, err := relayClient.RedeemBatch(conditionIds, negRisks, amounts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("result: %+v", result)
}

func TestSplitTokens(t *testing.T) {
	relayClient := &relayer.RelayClient{}

	_, err := relayClient.SplitTokens("", "1", false)
	if err == nil || err.Error() != "conditionId is empty" {
		t.Fatalf("SplitTokens() error = %v, want %q", err, "conditionId is empty")
	}

	_, err = relayClient.SplitTokens("0x123", "", false)
	if err == nil || err.Error() != "amount invalid" {
		t.Fatalf("SplitTokens() error = %v, want %q", err, "amount invalid")
	}

	_, err = relayClient.SplitTokens("0x123", "abc", false)
	if err == nil || !strings.Contains(err.Error(), "amount invalid") {
		t.Fatalf("SplitTokens() error = %v, want contains %q", err, "amount invalid")
	}
}

func TestMergeTokens(t *testing.T) {
	relayClient := &relayer.RelayClient{}

	_, err := relayClient.MergeTokens("", "1", false)
	if err == nil || err.Error() != "conditionId is empty" {
		t.Fatalf("MergeTokens() error = %v, want %q", err, "conditionId is empty")
	}

	_, err = relayClient.MergeTokens("0x123", "", false)
	if err == nil || err.Error() != "amount invalid" {
		t.Fatalf("MergeTokens() error = %v, want %q", err, "amount invalid")
	}

	_, err = relayClient.MergeTokens("0x123", "abc", false)
	if err == nil || !strings.Contains(err.Error(), "amount invalid") {
		t.Fatalf("MergeTokens() error = %v, want contains %q", err, "amount invalid")
	}
}

func TestSplitAndMergeLive(t *testing.T) {
	config := polymarket.DefaultConfig()
	config.Polymarket.BuilderCreds = &model.ApiKeyCreds{
		Key:        os.Getenv("BUILDER_API_KEY"),
		Secret:     os.Getenv("BUILDER_SECRET"),
		Passphrase: os.Getenv("BUILDER_PASSPHRASE"),
	}
	config.Polymarket.OwnerKey = os.Getenv("SIGNERKEY")
	conditionID := os.Getenv("CONDITIONID")
	if conditionID == "" {
		t.Fatal("CONDITIONID is required")
	}

	relayClient := relayer.NewRelayClient(config.Polymarket.RelayerBaseURL, config.Polymarket.OwnerKey, 137, config.Polymarket.BuilderCreds, nil, config.Polymarket.RelayerKey)

	// splitResult, err := relayClient.SplitTokens(conditionID, "1", false)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("split result: %+v", splitResult)

	mergeResult, err := relayClient.MergeTokens(conditionID, "1", false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("merge result: %+v", mergeResult)
}
