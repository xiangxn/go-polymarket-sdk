package eip712

import (
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	_EIP712_DOMAIN_HASH = crypto.Keccak256Hash(
		[]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
	)
)
