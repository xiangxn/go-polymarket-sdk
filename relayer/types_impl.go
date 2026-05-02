package relayer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	pgc "github.com/ivanzzeth/polymarket-go-contracts/v2"
	"github.com/xiangxn/go-polymarket-sdk/constants"
)

func DefaultSafeContractConfig(chainId int64) *SafeContractConfig {
	c := pgc.GetContractConfig(big.NewInt(chainId))
	return &SafeContractConfig{
		SafeFactory:   c.SafeProxyFactory,
		SafeMultisend: common.HexToAddress(constants.SAFE_MULTISEND),
	}
}
