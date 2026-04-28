package orders

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type Contracts struct {
	Exchange        common.Address
	NegRiskExchange common.Address
	NegRiskAdapter  common.Address
	Collateral      common.Address
	Conditional     common.Address
}

var (
	_POLYGON_CONTRACTS = &Contracts{
		Exchange:        common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B"),
		NegRiskExchange: common.HexToAddress("0xe2222d279d744050d28e00520010520000310F59"),
		NegRiskAdapter:  common.HexToAddress("0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"),
		Collateral:      common.HexToAddress("0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"),
		Conditional:     common.HexToAddress("0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"),
	}
)

func GetContracts(chainId int64) (*Contracts, error) {
	switch chainId {
	case 137:
		return _POLYGON_CONTRACTS, nil
	default:
		return nil, fmt.Errorf("invalid chain id")
	}
}
