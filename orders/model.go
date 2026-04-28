package orders

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type OrderSignature = []byte

type OrderHash = common.Hash

type OrderData struct {
	/**
	 * Maker of the order, i.e the source of funds for the order
	 */
	Maker string

	/**
	 * Token Id of the CTF ERC1155 asset to be bought or sold.
	 * If BUY, this is the tokenId of the asset to be bought, i.e the makerAssetId
	 * If SELL, this is the tokenId of the asset to be sold, i.e the  takerAssetId
	 */
	TokenId string

	/**
	 * Maker amount, i.e the max amount of tokens to be sold
	 */
	MakerAmount string

	/**
	 * Taker amount, i.e the minimum amount of tokens to be received
	 */
	TakerAmount string

	/**
	 * The side of the order, BUY or SELL
	 */
	Side Side

	/**
	 * Signer of the order. Optional, if it is not present the signer is the maker of the order.
	 */
	Signer *string

	/**
	 * Signature type used by the Order. Default value 'EOA'
	 */
	SignatureType *SignatureType

	/**
	 * Timestamp of the order
	 */
	Timestamp *string

	/**
	 * Metadata of the order
	 */
	Metadata *string

	/**
	 * Builder of the order
	 */
	Builder *string

	/**
	 * Expiration timestamp of the order (unix seconds, "0" = no expiration)
	 */
	Expiration *string
}

type Order struct {
	//  Unique salt to ensure entropy
	Salt *big.Int

	// Maker of the order, i.e the source of funds for the order
	Maker common.Address

	// Signer of the order
	Signer common.Address

	// Token Id of the CTF ERC1155 asset to be bought or sold.
	// If BUY, this is the tokenId of the asset to be bought, i.e the makerAssetId
	// If SELL, this is the tokenId of the asset to be sold, i.e the  takerAssetId
	TokenId *big.Int

	// Maker amount, i.e the max amount of tokens to be sold
	MakerAmount *big.Int

	// Taker amount, i.e the minimum amount of tokens to be received
	TakerAmount *big.Int

	// The side of the order, BUY or SELL
	Side *big.Int

	// Signature type used by the Order
	SignatureType *big.Int

	// Timestamp of the order
	Timestamp *big.Int

	// Metadata of the order
	Metadata common.Hash

	// Builder of the order
	Builder common.Hash
}

type SignedOrder struct {
	Order

	// The order signature
	Signature OrderSignature
}
