package orders

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	etheip712 "github.com/ivanzzeth/ethsig/eip712"
	"github.com/xiangxn/go-polymarket-sdk/signature"
)

type OrderBuilderImpl struct {
	chainId       *big.Int
	saltGenerator func() int64
}

var _ OrderBuilder = (*OrderBuilderImpl)(nil)

func NewOrderBuilderImpl(chainId *big.Int, saltGenerator func() int64) *OrderBuilderImpl {
	if saltGenerator == nil {
		saltGenerator = GenerateRandomSalt
	}
	return &OrderBuilderImpl{
		chainId:       chainId,
		saltGenerator: saltGenerator,
	}
}

// build an order object including the signature.
//
// @param private key
//
// @param orderData
//
// @returns a SignedOrder object (order + signature)
func (e *OrderBuilderImpl) BuildSignedOrder(privateKey *ecdsa.PrivateKey, orderData *OrderData, contract VerifyingContract) (*SignedOrder, error) {
	order, err := e.BuildOrder(orderData)
	if err != nil {
		log.Printf("BuildOrder: %+v", err)
		return nil, err
	}
	// log.Printf("order: %+v", order)

	orderHash, err := e.BuildOrderHash(order, contract)
	if err != nil {
		return nil, err
	}
	// log.Printf("orderHash: %s", orderHash.Hex())

	sign, err := e.BuildOrderSignature(privateKey, orderHash)
	if err != nil {
		return nil, err
	}
	// log.Printf("sign: %s", common.Bytes2Hex(sign))

	ok, err := signature.ValidateSignature(order.Signer, orderHash, sign)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("signature error")
	}

	return &SignedOrder{
		Order:     *order,
		Signature: sign,
	}, nil
}

// Creates an Order object from order data.
//
// @param orderData
//
// @returns a Order object (not signed)
func (e *OrderBuilderImpl) BuildOrder(orderData *OrderData) (*Order, error) {
	var signer common.Address
	if orderData.Signer == nil {
		signer = common.HexToAddress(orderData.Maker)
	} else {
		signer = common.HexToAddress(*orderData.Signer)
	}

	var tokenId *big.Int
	var ok bool
	if tokenId, ok = new(big.Int).SetString(orderData.TokenId, 10); !ok {
		return nil, fmt.Errorf("can't parse TokenId: %s as valid *big.Int", orderData.TokenId)
	}

	var makerAmount *big.Int
	if makerAmount, ok = new(big.Int).SetString(orderData.MakerAmount, 10); !ok {
		return nil, fmt.Errorf("can't parse MakerAmount: %s as valid *big.Int", orderData.MakerAmount)
	}

	var takerAmount *big.Int
	if takerAmount, ok = new(big.Int).SetString(orderData.TakerAmount, 10); !ok {
		return nil, fmt.Errorf("can't parse TakerAmount: %s as valid *big.Int", orderData.TakerAmount)
	}

	signatureType := EOA
	if orderData.SignatureType != nil {
		signatureType = *orderData.SignatureType
	}

	var timestamp *big.Int
	if orderData.Timestamp == nil {
		timestamp = big.NewInt(time.Now().UnixMilli())
	} else {
		if timestamp, ok = new(big.Int).SetString(*orderData.Timestamp, 10); !ok {
			return nil, fmt.Errorf("can't parse Timestamp: %s as valid *big.Int", *orderData.Timestamp)
		}
	}

	var metadata common.Hash
	if orderData.Metadata == nil {
		metadata = common.Hash{}
	} else {
		metadata = common.HexToHash(*orderData.Metadata)
	}

	var builder common.Hash
	if orderData.Builder == nil {
		builder = common.HexToHash("0x05218f8fe2ecac33c25ead880759a221303d7623f807be8b876a0bff9dd18b7c")
	} else {
		builder = common.HexToHash(*orderData.Builder)
	}

	side := 0
	if orderData.Side == SELL {
		side = 1
	}

	return &Order{
		Salt:          big.NewInt(e.saltGenerator()),
		Maker:         common.HexToAddress(orderData.Maker),
		Signer:        signer,
		TokenId:       tokenId,
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Side:          big.NewInt(int64(side)),
		SignatureType: big.NewInt(int64(signatureType)),
		Timestamp:     timestamp,
		Metadata:      metadata,
		Builder:       builder,
	}, nil
}

// Generates the hash of the order from a EIP712TypedData object.
//
// @param Order
//
// @returns a OrderHash that is a 'common.Hash'
func (e *OrderBuilderImpl) BuildOrderHash(order *Order, contract VerifyingContract) (OrderHash, error) {
	verifyingContract, err := GetVerifyingContractAddress(e.chainId, contract)
	if err != nil {
		return OrderHash{}, err
	}

	typedData := etheip712.TypedData{
		Types:       _ORDER_EIP712_TYPES,
		PrimaryType: _ORDER_PRIMARY_TYPE,
		Domain: etheip712.TypedDataDomain{
			Name:              _PROTOCOL_NAME,
			Version:           _PROTOCOL_VERSION,
			ChainId:           e.chainId.String(),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: etheip712.TypedDataMessage{
			"salt":          order.Salt,
			"maker":         order.Maker.Hex(),
			"signer":        order.Signer.Hex(),
			"tokenId":       order.TokenId,
			"makerAmount":   order.MakerAmount,
			"takerAmount":   order.TakerAmount,
			"side":          order.Side,
			"signatureType": order.SignatureType,
			"timestamp":     order.Timestamp,
			"metadata":      order.Metadata,
			"builder":       order.Builder,
		},
	}

	hash, _, err := etheip712.TypedDataAndHash(typedData)
	if err != nil {
		return OrderHash{}, err
	}

	return common.BytesToHash(hash), nil
}

// signs an order
//
// @param private key
//
// @param order hash
//
// @returns a OrderSignature that is []byte
func (e *OrderBuilderImpl) BuildOrderSignature(privateKey *ecdsa.PrivateKey, orderHash OrderHash) (OrderSignature, error) {
	return signature.Sign(privateKey, orderHash)
}
