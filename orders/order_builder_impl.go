package orders

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ivanzzeth/ethsig/eip712"
	"github.com/xiangxn/go-polymarket-sdk/signature"
)

type OrderBuilderImpl struct {
	chainId       *big.Int
	saltGenerator func() int64

	appDomainSeps map[string]common.Hash
}

var _ OrderBuilder = (*OrderBuilderImpl)(nil)

func NewOrderBuilderImpl(chainId *big.Int, saltGenerator func() int64) *OrderBuilderImpl {
	if saltGenerator == nil {
		saltGenerator = GenerateRandomSalt
	}
	return &OrderBuilderImpl{
		chainId:       chainId,
		saltGenerator: saltGenerator,
		appDomainSeps: make(map[string]common.Hash),
	}
}

func (e *OrderBuilderImpl) buildAppDomainSep(verifyingContract *common.Address) (common.Hash, error) {
	appHash, exists := e.appDomainSeps[verifyingContract.Hex()]
	if exists {
		return appHash, nil
	}
	args := abi.Arguments{
		{Type: MustType("bytes32")},
		{Type: MustType("bytes32")},
		{Type: MustType("bytes32")},
		{Type: MustType("uint256")},
		{Type: MustType("address")},
	}
	packed, err := args.Pack(
		_DOMAIN_TYPE_HASH,
		_PROTOCOL_NAME_HASH,
		_PROTOCOL_VERSION_HASH,
		e.chainId,
		*verifyingContract,
	)
	if err != nil {
		return common.Hash{}, err
	}
	appHash = crypto.Keccak256Hash(packed)
	e.appDomainSeps[verifyingContract.Hex()] = appHash
	return appHash, nil
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

	sign, err := e.BuildOrderSignature(privateKey, order, contract)
	if err != nil {
		return nil, err
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

	expiration := "0"
	if orderData.Expiration != nil {
		expiration = *orderData.Expiration
		if expiration == "" {
			expiration = "0"
		}
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
		Metadata:      metadata,
		Builder:       builder,
		Timestamp:     timestamp,
		Expiration:    &expiration,
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

	typedData := eip712.TypedData{
		Types:       _ORDER_EIP712_TYPES,
		PrimaryType: _ORDER_PRIMARY_TYPE,
		Domain: eip712.TypedDataDomain{
			Name:              _PROTOCOL_NAME,
			Version:           _PROTOCOL_VERSION,
			ChainId:           e.chainId.String(),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: eip712.TypedDataMessage{
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

	hash, _, err := eip712.TypedDataAndHash(typedData)
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
func (e *OrderBuilderImpl) BuildOrderSignature(privateKey *ecdsa.PrivateKey, order *Order, contract VerifyingContract) (OrderSignature, error) {
	if order.SignatureType.Cmp(big.NewInt(int64(POLY_1271))) == 0 {
		return e.BuildOrderSignature1271(privateKey, order, contract)
	}

	orderHash, err := e.BuildOrderHash(order, contract)
	if err != nil {
		return nil, err
	}

	sign, err := signature.Sign(privateKey, orderHash)
	if err != nil {
		return nil, err
	}

	ok, err := signature.ValidateSignature(order.Signer, orderHash, sign)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("signature error")
	}
	return sign, nil
}

var (
	_PROTOCOL_NAME_HASH    = crypto.Keccak256Hash([]byte(_PROTOCOL_NAME))
	_PROTOCOL_VERSION_HASH = crypto.Keccak256Hash([]byte(_PROTOCOL_VERSION))
	_ORDER_TYPE_HASH       = crypto.Keccak256Hash([]byte(_ORDER_TYPE_STRING))
	_DOMAIN_TYPE_HASH      = crypto.Keccak256Hash([]byte(_DOMAIN_TYPE_STRING))
)

func (e *OrderBuilderImpl) BuildOrderSignature1271(privateKey *ecdsa.PrivateKey, order *Order, contract VerifyingContract) (OrderSignature, error) {
	verifyingContract, err := GetVerifyingContractAddress(e.chainId, contract)
	if err != nil {
		return nil, err
	}

	appDomainSep, err := e.buildAppDomainSep(&verifyingContract)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments{
		{Type: MustType("bytes32")},
		{Type: MustType("uint256")},
		{Type: MustType("address")},
		{Type: MustType("address")},
		{Type: MustType("uint256")},
		{Type: MustType("uint256")},
		{Type: MustType("uint256")},
		{Type: MustType("uint8")},
		{Type: MustType("uint8")},
		{Type: MustType("uint256")},
		{Type: MustType("bytes32")},
		{Type: MustType("bytes32")},
	}
	packed, err := args.Pack(
		_ORDER_TYPE_HASH,  // [32]byte
		order.Salt,        // *big.Int
		order.Maker,       // common.Address
		order.Signer,      // common.Address
		order.TokenId,     // *big.Int
		order.MakerAmount, // *big.Int
		order.TakerAmount, // *big.Int
		uint8(order.Side.Int64()),
		uint8(order.SignatureType.Int64()),
		order.Timestamp, // *big.Int
		order.Metadata,  // [32]byte
		order.Builder,   // [32]byte
	)
	if err != nil {
		return nil, err
	}
	contentsHash := crypto.Keccak256Hash(packed)

	typedData := eip712.TypedData{
		Types:       _ORDER_EIP712_TYPES_1271,
		PrimaryType: "TypedDataSign",
		Domain: eip712.TypedDataDomain{
			Name:              _PROTOCOL_NAME,
			Version:           _PROTOCOL_VERSION,
			ChainId:           e.chainId.String(),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: eip712.TypedDataMessage{
			"contents": eip712.TypedDataMessage{
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
			"name":              "DepositWallet",
			"version":           "1",
			"chainId":           e.chainId.String(),
			"verifyingContract": order.Signer.Hex(),
			"salt":              common.Hash{},
		},
	}
	hash, _, err := eip712.TypedDataAndHash(typedData)
	if err != nil {
		return nil, err
	}
	// log.Printf("typedDataHash: %s", common.BytesToHash(hash).Hex())

	innerSig, err := signature.Sign(privateKey, common.BytesToHash(hash))

	ctLen := len(_ORDER_TYPE_STRING)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(ctLen))

	contentsType := []byte(_ORDER_TYPE_STRING)
	totalLen := len(innerSig) + len(appDomainSep) + len(contentsHash) + len(contentsType) + 2

	result := make([]byte, totalLen)
	offset := 0
	offset += copy(result[offset:], innerSig)
	offset += copy(result[offset:], appDomainSep[:])
	offset += copy(result[offset:], contentsHash[:])
	offset += copy(result[offset:], contentsType[:])
	copy(result[offset:], lenBuf)

	// log.Printf("innerSig: %s", hexutil.Encode(innerSig))
	// log.Printf("appDomainSep: %s", appDomainSep.Hex())
	// log.Printf("contentsHash: %s", contentsHash.Hex())
	// log.Printf("contentsType: %s", hexutil.Encode(contentsType))
	// log.Printf("lenBuf: %s", hexutil.Encode(lenBuf))

	// log.Printf("result: %s", hexutil.Encode(result))
	return result, nil
}
