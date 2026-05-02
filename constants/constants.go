package constants

import (
	"github.com/ethereum/go-ethereum/common"
)

const CollateralTokenDecimals = 6

const (
	SAFE_MULTISEND       = "0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761"
	SAFE_INIT_CODE_HASH  = "0x2bce2127ff07fb632d16c8347c4ebf501f4841168bed00d9e6ef715ddb6fcecf"
	PROXY_INIT_CODE_HASH = "0xd21df8dc65880a8606f09fe0ce3df9b8869287ab0b058be05aa9e8af6330a00b"
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
	HashZero    = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
)
