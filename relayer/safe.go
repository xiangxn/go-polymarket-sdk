package relayer

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ivanzzeth/ethsig"
	"github.com/ivanzzeth/ethsig/eip712"
	pgc "github.com/ivanzzeth/polymarket-go-contracts/v2"
	"github.com/xiangxn/go-polymarket-sdk/constants"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func BuildSafeTransactionRequest(
	signer *polymarket.Signer,
	args *SafeTransactionArgs,
	safeContractConfig *SafeContractConfig,
) (*TransactionRequest, error) {
	transaction, err := AggregateTransaction(args.Transactions, safeContractConfig.SafeMultisend)
	if err != nil {
		return nil, err
	}
	safeTxnGas := big.NewInt(0)
	baseGas := big.NewInt(0)
	gasPrice := big.NewInt(0)
	gasToken := constants.ZeroAddress
	refundReceiver := constants.ZeroAddress
	safeAddress := DeriveSafe(args.From, safeContractConfig.SafeFactory)

	typedData := pgc.BuildSafeTransactionTypedData(args.ChainId, safeAddress, transaction.To, transaction.Value, transaction.Data, transaction.Operation, safeTxnGas, baseGas, gasPrice, gasToken, refundReceiver, args.Nonce)

	eoaSigner := ethsig.NewEthPrivateKeySigner(signer.PrivateKey)
	sig, err := SignTypedData(eoaSigner, typedData)

	sigParams := SignatureParams{
		GasPrice:       utils.StringPtr(gasPrice.String()),
		Operation:      utils.StringPtr(strconv.Itoa(int(transaction.Operation))),
		SafeTxnGas:     utils.StringPtr(safeTxnGas.String()),
		BaseGas:        utils.StringPtr(baseGas.String()),
		GasToken:       utils.StringPtr(gasToken.String()),
		RefundReceiver: utils.StringPtr(refundReceiver.String()),
	}
	return &TransactionRequest{
		From:            args.From.Hex(),
		To:              transaction.To.Hex(),
		ProxyWallet:     utils.StringPtr(safeAddress.Hex()),
		Data:            "0x" + common.Bytes2Hex(transaction.Data),
		Nonce:           utils.StringPtr(args.Nonce.String()),
		Signature:       "0x" + common.Bytes2Hex(sig),
		SignatureParams: sigParams,
		Type:            string(TT_SAFE),
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

func SignTypedData(signer *ethsig.EthPrivateKeySigner, typedData eip712.TypedData) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, ethsig.NewEIP712Error("failed to hash domain", err)
	}
	// fmt.Println("domainSeparator:", domainSeparator.String())
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, ethsig.NewEIP712Error("failed to hash message", err)
	}
	// fmt.Println("typedDataHash:", typedDataHash.String())
	// Create EIP-191 version 0x01 message
	// rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	rawData := fmt.Appendf(nil, "\x19\x01%s%s", string(domainSeparator), string(typedDataHash))
	digest := crypto.Keccak256Hash(rawData)
	// fmt.Println("structHash:", digest.String())

	signature, err := signer.PersonalSign(string(digest.Bytes()))
	if err != nil {
		return nil, ethsig.NewSignatureError("failed to sign", err)
	}

	last := signature[len(signature)-1]
	v := int(last)

	switch v {
	case 0, 1:
		signature[len(signature)-1] = last + 31
	case 27, 28:
		signature[len(signature)-1] = last + 4
	default:
		return nil, ethsig.NewSignatureError("invalid signature v value", nil)
	}

	// fmt.Println("signature:", common.Bytes2Hex(signature))
	return signature, nil
}
