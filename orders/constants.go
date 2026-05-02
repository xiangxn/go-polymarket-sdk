package orders

import "github.com/ivanzzeth/ethsig/eip712"

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
