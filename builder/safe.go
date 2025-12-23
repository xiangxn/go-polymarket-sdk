package builder

import (
	"bytes"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ivanzzeth/ethsig"
	pgc "github.com/ivanzzeth/polymarket-go-contracts"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func BuildSafeTransactionRequest(
	signer *polymarket.Signer,
	args *SafeTransactionArgs,
	safeContractConfig *SafeContractConfig,
	metadata *string,
) (*TransactionRequest, error) {
	transaction, err := AggregateTransaction(args.Transactions, safeContractConfig.SafeMultisend)
	if err != nil {
		return nil, err
	}
	safeTxnGas := big.NewInt(0)
	baseGas := big.NewInt(0)
	gasPrice := big.NewInt(0)
	gasToken := ZeroAddress
	refundReceiver := ZeroAddress
	safeAddress := DeriveSafe(args.From, safeContractConfig.SafeFactory)

	typedData := pgc.BuildSafeTransactionTypedData(args.ChainId, safeAddress, transaction.To, transaction.Value, transaction.Data, transaction.Operation, safeTxnGas, baseGas, gasPrice, gasToken, refundReceiver, args.Nonce)

	eoaSigner := ethsig.NewEthPrivateKeySigner(signer.PrivateKey)
	sig, err := eoaSigner.SignTypedData(typedData)

	sigParams := SignatureParams{
		GasPrice:       utils.StringPtr(gasPrice.String()),
		Operation:      utils.StringPtr(strconv.Itoa(int(transaction.Operation))),
		SafeTxnGas:     utils.StringPtr(safeTxnGas.String()),
		BaseGas:        utils.StringPtr(baseGas.String()),
		GasToken:       utils.StringPtr(gasToken.String()),
		RefundReceiver: utils.StringPtr(refundReceiver.String()),
	}
	if metadata == nil {
		metadata = utils.StringPtr("")
	}
	return &TransactionRequest{
		From:            args.From.Hex(),
		To:              transaction.To.Hex(),
		ProxyWallet:     utils.StringPtr(safeAddress.Hex()),
		Data:            string(transaction.Data),
		Nonce:           utils.StringPtr(args.Nonce.String()),
		Signature:       string(sig),
		SignatureParams: sigParams,
		Type:            string(TT_SAFE),
		Metadata:        metadata,
	}, nil
}

func AggregateTransaction(txns []SafeTransaction, safeMultisend common.Address) (*SafeTransaction, error) {
	var transaction *SafeTransaction
	if len(txns) == 1 {
		transaction = &txns[0]
	} else {
		tr, err := CreateSafeMultisendTransaction(txns, safeMultisend)
		if err != nil {
			return nil, err
		}
		transaction = tr
	}
	return transaction, nil
}

func CreateSafeMultisendTransaction(txns []SafeTransaction, safeMultisendAddress common.Address) (*SafeTransaction, error) {

	var packed bytes.Buffer

	for _, tx := range txns {
		// encodePacked(
		// ["uint8","address","uint256","uint256","bytes"],
		// [operation, to, value, data.length, data]
		// )

		packed.WriteByte(byte(tx.Operation)) // uint8
		packed.Write(tx.To.Bytes())          // address (20 bytes)
		// uint256 value (decimal string → big.Int)
		packed.Write(common.LeftPadBytes(tx.Value.Bytes(), 32))

		dataLen := big.NewInt(int64(len(tx.Data)))
		packed.Write(common.LeftPadBytes(dataLen.Bytes(), 32)) // uint256
		packed.Write(tx.Data)                                  // bytes
	}

	// multisend(bytes transactions)
	multisendABI, _ := abi.JSON(strings.NewReader(MultisendABIJSON))
	data, err := multisendABI.Pack("multiSend", packed.Bytes())
	if err != nil {
		return nil, err
	}

	return &SafeTransaction{
		To:        safeMultisendAddress,
		Value:     big.NewInt(0),
		Data:      data,
		Operation: pgc.SafeOperationDelegateCall,
	}, nil
}
