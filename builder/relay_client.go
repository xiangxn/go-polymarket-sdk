package builder

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	pgc "github.com/ivanzzeth/polymarket-go-contracts"
	ctokens "github.com/ivanzzeth/polymarket-go-contracts/contracts/conditional-tokens"
	negriskadapter "github.com/ivanzzeth/polymarket-go-contracts/contracts/neg-risk-adapter"
	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/headers"
	Headers "github.com/xiangxn/go-polymarket-sdk/headers"
	"github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"resty.dev/v3"
)

type RelayClient struct {
	relayerBaseURL     string
	chainId            *big.Int
	http               *resty.Client
	signer             *polymarket.Signer
	BuilderCreds       *model.ApiKeyCreds
	safeContractConfig *SafeContractConfig
}

func NewRelayClient(relayerUrl string, signerKey string, chainId int64, builderCreds *model.ApiKeyCreds, safeContractConfig *SafeContractConfig) *RelayClient {
	privateKey, err := crypto.HexToECDSA(signerKey)
	if err != nil {
		panic(err)
	}
	if safeContractConfig == nil {
		safeContractConfig = DefaultSafeContractConfig()
	}
	return &RelayClient{
		relayerBaseURL:     relayerUrl,
		http:               resty.New(),
		signer:             &polymarket.Signer{PrivateKey: privateKey, Address: crypto.PubkeyToAddress(privateKey.PublicKey)},
		BuilderCreds:       builderCreds,
		chainId:            new(big.Int).SetInt64(chainId),
		safeContractConfig: safeContractConfig,
	}
}

func (c *RelayClient) Get(url string, params map[string]string, headers map[string]string) (*gjson.Result, error) {
	request := c.http.R()
	if params != nil {
		request.SetQueryParams(params)
	}
	Headers.OverloadRelayHeaders(resty.MethodPost, headers)
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
func (c *RelayClient) Post(url string, body any, headers map[string]string) (*gjson.Result, error) {
	request := c.http.R()
	if body != nil {
		request.SetBody(body)
	}
	Headers.OverloadRelayHeaders(resty.MethodPost, headers)
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

func (c *RelayClient) GetNonce(signerAddress common.Address, signerType TransactionType) (*big.Int, error) {
	url := fmt.Sprintf("%s%s", c.relayerBaseURL, GET_NONCE)
	params := map[string]string{
		"address": signerAddress.Hex(),
		"type":    string(signerType),
	}
	result, err := c.Get(url, params, nil)
	if err != nil {
		return nil, err
	}
	nonceStr := result.Get("nonce").String()
	nonce, ok := new(big.Int).SetString(nonceStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse nonce: %s", nonceStr)
	}

	return nonce, nil
}

func (c *RelayClient) GetDeployed(safe common.Address) (bool, error) {
	url := fmt.Sprintf("%s%s", c.relayerBaseURL, GET_DEPLOYED)
	result, err := c.Get(url, map[string]string{"address": safe.Hex()}, nil)
	if err != nil {
		return false, err
	}
	deployed := result.Get("deployed").Bool()
	return deployed, nil
}

func (c *RelayClient) GetRelayPayload(signerAddress common.Address, signerType string) (*RelayPayload, error) {
	url := fmt.Sprintf("%s%s", c.relayerBaseURL, GET_RELAY_PAYLOAD)
	params := map[string]string{
		"address": signerAddress.Hex(),
		"type":    signerType,
	}
	result, err := c.Get(url, params, nil)
	if err != nil {
		return nil, err
	}
	return &RelayPayload{
		Address: result.Get("address").String(),
		Nonce:   result.Get("nonce").String(),
	}, nil
}

func (c *RelayClient) EexecuteSafeTransactions(txns []SafeTransaction, metadata *string) (*RelayerTransactionResponse, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}

	url := fmt.Sprintf("%s%s", c.relayerBaseURL, SUBMIT_TRANSACTION)
	safe := c.GetExpectedSafe(c.signer.Address)
	// log.Printf("safe: %s, signer: %s", safe.Hex(), c.signer.Address.Hex())
	deployed, err := c.GetDeployed(safe)
	if err != nil {
		return nil, err
	}
	if !deployed {
		return nil, fmt.Errorf("safe is not deployed")
	}

	from := c.signer.Address
	nonce, err := c.GetNonce(from, TT_SAFE)
	if err != nil {
		return nil, err
	}
	args := SafeTransactionArgs{
		Transactions: txns,
		From:         from,
		Nonce:        nonce,
		ChainId:      c.chainId,
	}
	if c.safeContractConfig == nil {
		return nil, fmt.Errorf("safeContractConfig is nil")
	}

	request, err := BuildSafeTransactionRequest(c.signer, &args, c.safeContractConfig, metadata)
	if err != nil {
		return nil, err
	}

	requestPayload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	body := string(requestPayload)
	// log.Printf("body: %s", body)
	var builderHeaders map[string]string
	if c.BuilderCreds != nil {
		builderHeaders = headers.CreateBuilderHeaders(c.BuilderCreds, resty.MethodPost, SUBMIT_TRANSACTION, &body, nil)
	}
	resp, err := c.Post(url, body, builderHeaders)
	if err != nil {
		return nil, err
	}
	return &RelayerTransactionResponse{
		TransactionID:   resp.Get("transactionID").String(),
		State:           resp.Get("state").String(),
		Hash:            resp.Get("hash").String(),
		TransactionHash: resp.Get("transactionHash").String(),
	}, nil
}

func (c *RelayClient) GetExpectedSafe(signer common.Address) common.Address {
	return DeriveSafe(signer, c.safeContractConfig.SafeFactory)
}

func (c *RelayClient) RedeemBatch(conditionIds []string, negRisks []bool, amounts [][]*big.Int, metadatas []any) ([]any, error) {
	if conditionIds == nil {
		return nil, fmt.Errorf("conditionIds is nil")
	}
	if len(conditionIds) != len(negRisks) || len(conditionIds) != len(amounts) {
		return nil, fmt.Errorf("conditionIds, negRisks, amounts length not match")
	}
	redeemTxs := []SafeTransaction{}
	negRiskABI, err := negriskadapter.NegRiskAdapterMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse NegRiskAdapter ABI: %w", err)
	}
	ctfABI, err := ctokens.ConditionalTokensMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ConditionalTokens ABI: %w", err)
	}
	indexSets := []*big.Int{big.NewInt(1), big.NewInt(2)}
	for i, conditionId := range conditionIds {
		negRisk := negRisks[i]
		if negRisk {
			ams := amounts[i]
			calldata, err2 := negRiskABI.Pack("redeemPositions", common.HexToHash(conditionId), ams)
			if err2 != nil {
				return nil, fmt.Errorf("failed to pack redeemPositions calldata: %w", err2)
			}
			redeemTxs = append(redeemTxs, SafeTransaction{
				To:        common.HexToAddress(NEG_RISK_CTF_ADDRESS),
				Operation: pgc.SafeOperationCall,
				Data:      calldata,
				Value:     big.NewInt(0),
			})
		} else {
			calldata, err3 := ctfABI.Pack("redeemPositions", common.HexToAddress(USDC_ADDRESS), HashZero, common.HexToHash(conditionId), indexSets)
			if err3 != nil {
				return nil, fmt.Errorf("failed to pack redeemPositions calldata: %w", err3)
			}
			redeemTxs = append(redeemTxs, SafeTransaction{
				To:        common.HexToAddress(CTF_ADDRESS),
				Operation: pgc.SafeOperationCall,
				Data:      calldata,
				Value:     big.NewInt(0),
			})
		}
	}

	meta := "Redeem batch position"
	if metadatas != nil {
		metadata, err0 := json.Marshal(metadatas)
		if err0 != nil {
			return nil, err0
		}
		meta = string(metadata)
	}

	resp, err := c.EexecuteSafeTransactions(redeemTxs, &meta)
	if err != nil {
		return nil, err
	}
	log.Printf("redeem: %+v", resp)
	return metadatas, nil
}
