package tests

import (
	"fmt"
	"testing"

	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestEncryptDecrypt(t *testing.T) {
	enc := polymarket.NewEncryptor("my-strong-password")

	cipherText, _ := enc.Encrypt("hello world")
	plainText, _ := enc.Decrypt(cipherText)

	fmt.Println(cipherText)
	fmt.Println(plainText)

}
