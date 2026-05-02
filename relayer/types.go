package relayer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	pgc "github.com/ivanzzeth/polymarket-go-contracts/v2"
)

type SafeTransaction struct {
	To        common.Address
	Operation pgc.SafeOperation
	Data      []byte
	Value     *big.Int
}

type RelayerTransactionResponse struct {
	TransactionID   string `json:"transactionID"`
	State           string `json:"state"`
	Hash            string `json:"hash"`
	TransactionHash string `json:"transactionHash"`
}

type RelayPayload struct {
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
}

type TransactionType string

const (
	TT_SAFE        TransactionType = "SAFE"
	TT_PROXY       TransactionType = "PROXY"
	TT_SAFE_CREATE TransactionType = "SAFE-CREATE"
)

type SafeTransactionArgs struct {
	From         common.Address
	Nonce        *big.Int
	ChainId      *big.Int
	Transactions []SafeTransaction
}

type SafeContractConfig struct {
	SafeFactory   common.Address
	SafeMultisend common.Address
}

type SignatureParams struct {
	GasPrice *string `json:"gasPrice,omitempty"`

	// Proxy RelayHub sig params
	RelayerFee *string `json:"relayerFee,omitempty"`
	// gasPrice: string; // User supplied minimum gas price
	GasLimit *string `json:"gasLimit,omitempty"` // User supplied gas limit
	RelayHub *string `json:"relayHub,omitempty"` // Relay Hub Address
	Relay    *string `json:"relay,omitempty"`    // Relayer address

	// SAFE sig parameters
	Operation  *string `json:"operation,omitempty"`
	SafeTxnGas *string `json:"safeTxnGas,omitempty"`
	BaseGas    *string `json:"baseGas,omitempty"`
	// gasPrice: string;
	GasToken       *string `json:"gasToken,omitempty"`
	RefundReceiver *string `json:"refundReceiver,omitempty"`

	// SAFE CREATE sig parameters
	PaymentToken    *string `json:"paymentToken,omitempty"`
	Payment         *string `json:"payment,omitempty"`
	PaymentReceiver *string `json:"paymentReceiver,omitempty"`
}

type TransactionRequest struct {
	From            string          `json:"from"`
	To              string          `json:"to"`
	ProxyWallet     *string         `json:"proxyWallet,omitempty"`
	Data            string          `json:"data"`
	Nonce           *string         `json:"nonce,omitempty"`
	Signature       string          `json:"signature"`
	SignatureParams SignatureParams `json:"signatureParams"`
	Type            string          `json:"type"`
}
