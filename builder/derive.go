package builder

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func DeriveProxyWallet(address common.Address, proxyFactory common.Address) common.Address {

	// encodePacked(address) → 20 bytes
	packed := address.Bytes()

	saltBytes := crypto.Keccak256(packed)

	var salt [32]byte
	copy(salt[:], saltBytes)

	return GetCreate2Address(
		proxyFactory,
		salt,
		common.HexToHash(PROXY_INIT_CODE_HASH).Bytes(),
	)
}

func DeriveSafe(address common.Address, safeFactory common.Address) common.Address {

	// abi.encode(address) → 32 bytes left-padded
	encoded := make([]byte, 32)
	copy(encoded[12:], address.Bytes())

	saltBytes := crypto.Keccak256(encoded)

	var salt [32]byte
	copy(salt[:], saltBytes)

	return GetCreate2Address(
		safeFactory,
		salt,
		common.HexToHash(SAFE_INIT_CODE_HASH).Bytes(),
	)
}
