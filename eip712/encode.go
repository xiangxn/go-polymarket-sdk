package eip712

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func Encode(args []abi.Type, values []any) ([]byte, error) {
	arguments := make([]abi.Argument, 0)
	for _, t := range args {
		argument := abi.Argument{Type: t}
		arguments = append(arguments, argument)
	}

	return abi.Arguments(arguments).Pack(values...)
}

func pad32(b []byte) []byte {
	if len(b) == 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}
