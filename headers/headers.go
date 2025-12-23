package headers

import (
	"strconv"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/signature"
)

func CreateL2Headers(signer string, creds *ApiKeyCreds, l2HeaderArgs *L2HeaderArgs, timestamp *int64) map[string]string {
	if timestamp == nil {
		now := time.Now().Unix()
		timestamp = &now
	}
	if creds == nil {
		panic("creds is nil")
	}
	if l2HeaderArgs == nil {
		panic("l2HeaderArgs is nil")
	}
	signature := signature.GenSignature(creds.Secret, *timestamp, l2HeaderArgs.Method, l2HeaderArgs.RequestPath, l2HeaderArgs.Body)
	return map[string]string{
		"POLY_ADDRESS":    signer,
		"POLY_SIGNATURE":  signature,
		"POLY_TIMESTAMP":  strconv.FormatInt(*timestamp, 10),
		"POLY_API_KEY":    creds.Key,
		"POLY_PASSPHRASE": creds.Passphrase,
	}
}

func CreateBuilderHeaders(builderCreds *ApiKeyCreds, method string, path string, body *string, timestamp *int64) map[string]string {
	if timestamp == nil {
		now := time.Now().Unix()
		timestamp = &now
	}
	if builderCreds == nil {
		panic("builderCreds is nil")
	}
	signature := signature.GenSignature(builderCreds.Secret, *timestamp, method, path, body)
	return map[string]string{
		"POLY_BUILDER_API_KEY":    builderCreds.Key,
		"POLY_BUILDER_PASSPHRASE": builderCreds.Passphrase,
		"POLY_BUILDER_SIGNATURE":  signature,
		"POLY_BUILDER_TIMESTAMP":  strconv.FormatInt(*timestamp, 10),
	}
}
