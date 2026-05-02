package orders

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	pgc "github.com/ivanzzeth/polymarket-go-contracts/v2"
)

type OrderBuilder interface {
	// build an order object including the signature.
	//
	// @param private key
	//
	// @param orderData
	//
	// @returns a SignedOrder object (order + signature)
	BuildSignedOrder(privateKey *ecdsa.PrivateKey, orderData *OrderData, contract VerifyingContract) (*SignedOrder, error)

	// Creates an Order object from order data.
	//
	// @param orderData
	//
	// @returns a Order object (not signed)
	BuildOrder(orderData *OrderData) (*Order, error)

	// Generates the hash of the order from a EIP712TypedData object.
	//
	// @param Order
	//
	// @returns a OrderHash that is a 'common.Hash'
	BuildOrderHash(order *Order, contract VerifyingContract) (OrderHash, error)

	// signs an order
	//
	// @param private key
	//
	// @param order hash
	//
	// @returns a OrderSignature that is []byte
	BuildOrderSignature(privateKey *ecdsa.PrivateKey, orderHash OrderHash) (OrderSignature, error)
}

func GetVerifyingContractAddress(chainId *big.Int, contract VerifyingContract) (common.Address, error) {
	contracts := pgc.GetContractConfig(chainId)

	switch contract {
	case CTFExchange:
		return contracts.ExchangeV2, nil
	case NegRiskCTFExchange:
		return contracts.NegRiskExchangeV2, nil
	}

	return common.Address{}, fmt.Errorf("invalid contract")
}
