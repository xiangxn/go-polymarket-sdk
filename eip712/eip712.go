package eip712

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type domainKey struct {
	Name    string
	Version string
	ChainId string // 用 string 避免 big.Int 作为 key 的坑
	Addr    common.Address
}

var domainCache sync.Map // map[domainKey]common.Hash

func BuildEIP712DomainSeparator(name, version common.Hash, chainId *big.Int, verifyingContract common.Address) common.Hash {
	key := domainKey{
		Name:    name.Hex(),
		Version: version.Hex(),
		ChainId: chainId.String(),
		Addr:    verifyingContract,
	}

	// fast path
	if v, ok := domainCache.Load(key); ok {
		return v.(common.Hash)
	}

	// 计算
	ds := HashDomain(name, version, chainId, verifyingContract)

	// 避免重复写（并发安全）
	actual, _ := domainCache.LoadOrStore(key, ds)
	return actual.(common.Hash)
}

func HashDomain(name, version common.Hash, chainId *big.Int, verifyingContract common.Address) common.Hash {

	buf := make([]byte, 0, 32*5)

	buf = append(buf, _EIP712_DOMAIN_HASH.Bytes()...)
	buf = append(buf, name.Bytes()...)
	buf = append(buf, version.Bytes()...)
	buf = append(buf, pad32(chainId.Bytes())...)
	buf = append(buf, pad32(verifyingContract.Bytes())...)

	return crypto.Keccak256Hash(buf)
}

func HashTypedData(domainSeparator common.Hash, args []abi.Type, values []any) (common.Hash, error) {
	encoded, err := Encode(args, values)
	if err != nil {
		return common.Hash{}, err
	}

	hash := crypto.Keccak256Hash(encoded).Bytes()

	rawData := make([]byte, 0, 66)
	rawData = append(rawData, 0x19, 0x01)
	rawData = append(rawData, domainSeparator[:]...)
	rawData = append(rawData, hash...)
	return crypto.Keccak256Hash(rawData), nil
}
