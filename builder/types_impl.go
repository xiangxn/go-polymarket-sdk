package builder

import "github.com/ethereum/go-ethereum/common"

func DefaultSafeContractConfig() *SafeContractConfig {
	return &SafeContractConfig{
		SafeFactory:   common.HexToAddress(SAFE_FACTORY),
		SafeMultisend: common.HexToAddress(SAFE_MULTISEND),
	}
}
