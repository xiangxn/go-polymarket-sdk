package orders

import (
	"github.com/ivanzzeth/ethsig/eip712"
)

const (
	_PROTOCOL_NAME    = "Polymarket CTF Exchange"
	_PROTOCOL_VERSION = "2"
)

const _ORDER_PRIMARY_TYPE = "Order"

var _ORDER_EIP712_TYPES = eip712.Types{
	"EIP712Domain": {
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
	},
	_ORDER_PRIMARY_TYPE: {
		{Name: "salt", Type: "uint256"},
		{Name: "maker", Type: "address"},
		{Name: "signer", Type: "address"},
		{Name: "tokenId", Type: "uint256"},
		{Name: "makerAmount", Type: "uint256"},
		{Name: "takerAmount", Type: "uint256"},
		{Name: "side", Type: "uint8"},
		{Name: "signatureType", Type: "uint8"},
		{Name: "timestamp", Type: "uint256"},
		{Name: "metadata", Type: "bytes32"},
		{Name: "builder", Type: "bytes32"},
	},
}

var _ORDER_EIP712_TYPES_1271 = eip712.Types{
	"EIP712Domain": _ORDER_EIP712_TYPES["EIP712Domain"],
	"TypedDataSign": {
		{Name: "contents", Type: _ORDER_PRIMARY_TYPE},
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
		{Name: "salt", Type: "bytes32"},
	},
	_ORDER_PRIMARY_TYPE: _ORDER_EIP712_TYPES[_ORDER_PRIMARY_TYPE],
}

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

type VerifyingContract = int

const (
	CTFExchange VerifyingContract = iota
	NegRiskCTFExchange
)

const (
	_ORDER_TYPE_STRING  = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"
	_DOMAIN_TYPE_STRING = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
)
