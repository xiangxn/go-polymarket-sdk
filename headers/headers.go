package headers

import (
	"maps"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-polymarket-sdk/signature"
	"resty.dev/v3"
)

func CreateL2Headers(signer common.Address, creds *ApiKeyCreds, l2HeaderArgs *L2HeaderArgs, timestamp *int64) map[string]string {
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
		"POLY_ADDRESS":    signer.Hex(),
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

func OverloadHeaders(method string, headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}
	maps.Copy(headers, map[string]string{
		"User-Agent":   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
		"Accept":       "*/*",
		"Connection":   "keep-alive",
		"Content-Type": "application/json",
	})
	if method == resty.MethodGet || method == resty.MethodPost {
		maps.Copy(headers, map[string]string{
			"Accept-Encoding": "gzip",
		})
	}
}

func OverloadRelayHeaders(method string, headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}
	maps.Copy(headers, map[string]string{
		"User-Agent":                       "@polymarket/relay-client",
		"Accept":                           "*/*",
		"Connection":                       "keep-alive",
		"Content-Type":                     "application/json",
		"Access-Control-Allow-Credentials": "true",
	})
	if method == resty.MethodGet {
		maps.Copy(headers, map[string]string{
			"Accept-Encoding": "gzip",
		})
	}
}
