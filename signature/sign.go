package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"strconv"
	"strings"
)

// BuildHmacSignature builds an HMAC signature
// secret: base64 encoded secret key
// timestamp: Unix timestamp
// method: HTTP method (e.g., "POST", "GET")
// requestPath: API endpoint path
// body: Optional request body
// Returns: URL-safe base64 encoded HMAC signature
func GenSignature(
	secret string,
	timestamp int64,
	method string,
	requestPath string,
	body *string,
) string {
	// Build the message: timestamp + method + path + body (if present)
	message := strconv.FormatInt(timestamp, 10) + method + requestPath
	if body != nil {
		message += *body
	}
	log.Println("message: ", message)
	// Decode the base64 secret
	base64Secret, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		// If decoding fails, treat secret as raw bytes
		base64Secret = []byte(secret)
	}
	log.Println("base64Secret: ", base64Secret)

	// Create HMAC-SHA256
	h := hmac.New(sha256.New, base64Secret)
	h.Write([]byte(message))
	sig := h.Sum(nil)

	// Encode to base64
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	// Convert to URL-safe base64: '+' -> '-', '/' -> '_'
	// Keep '=' padding as is
	sigURLSafe := strings.ReplaceAll(sigBase64, "+", "-")
	sigURLSafe = strings.ReplaceAll(sigURLSafe, "/", "_")

	return sigURLSafe
}
