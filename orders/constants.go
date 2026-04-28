package orders

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/xiangxn/go-polymarket-sdk/eip712"
)

var (
	_PROTOCOL_NAME    = crypto.Keccak256Hash([]byte("Polymarket CTF Exchange"))
	_PROTOCOL_VERSION = crypto.Keccak256Hash([]byte("2"))
)

var (
	_ORDER_STRUCTURE = []abi.Type{
		eip712.Bytes32, // typehash
		eip712.Uint256, // salt
		eip712.Address, // maker
		eip712.Address, // signer
		eip712.Uint256, // tokenId
		eip712.Uint256, // makerAmount
		eip712.Uint256, // takerAmount
		eip712.Uint8,   // side
		eip712.Uint8,   // signatureType
		eip712.Uint256, // timestamp
		eip712.Bytes32, // metadata
		eip712.Bytes32, // builder
	}
)

var (
	_ORDER_STRUCTURE_HASH = crypto.Keccak256Hash(
		[]byte("Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder,bytes signature)"),
	)
)

type SignatureType = int

const (
	/**
	 * ECDSA EIP712 signatures signed by EOAs
	 */
	EOA SignatureType = iota

	/**
	 * EIP712 signatures signed by EOAs that own Polymarket Proxy wallets
	 */
	POLY_PROXY

	/**
	 * EIP712 signatures signed by EOAs that own Polymarket Gnosis safes
	 */
	POLY_GNOSIS_SAFE

	/**
	 * EIP1271 signatures signed by smart contracts. To be used by smart contract wallets or vaults
	 */
	POLY_1271
)

type Side = int

const (
	BUY Side = iota
	SELL
)

type VerifyingContract = int

const (
	CTFExchange VerifyingContract = iota
	NegRiskCTFExchange
)
