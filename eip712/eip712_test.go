package eip712

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestHashDomain_WithGoOrderUtils(t *testing.T) {
	name := crypto.Keccak256Hash([]byte("Polymarket CTF Exchange"))
	version := crypto.Keccak256Hash([]byte("1"))
	chainID := big.NewInt(137)
	verifyingContract := common.HexToAddress("0x4bFb41d5B3570deFd03C39a9A4d8dE6bd8B8982E")

	testDomainSeparator := common.HexToHash("0x1a573e3617c78403b5b4b892827992f027b03d4eaf570048b8ee8cdd84d151be")

	t.Logf("test  HashDomain: %s", testDomainSeparator.Hex())

	hashDomain := HashDomain(name, version, chainID, verifyingContract)
	t.Logf("local HashDomain: %s", hashDomain.Hex())

	if hashDomain != testDomainSeparator {
		t.Fatalf("domain separator mismatch, local=%s, go-order-utils=%s", hashDomain.Hex(), testDomainSeparator.Hex())
	}
}
