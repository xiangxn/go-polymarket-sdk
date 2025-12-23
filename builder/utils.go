package builder

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetCreate2Address replicates viem / Solidity CREATE2 address calculation
func GetCreate2Address(
	deployer common.Address,
	salt [32]byte,
	initCode []byte,
) common.Address {
	initCodeHash := crypto.Keccak256(initCode)

	data := []byte{0xff}
	data = append(data, deployer.Bytes()...)
	data = append(data, salt[:]...)
	data = append(data, initCodeHash...)

	hash := crypto.Keccak256(data)
	return common.BytesToAddress(hash[12:])
}
