package headers

type L2HeaderArgs struct {
	Method      string
	RequestPath string
	Body        *string
}

type ApiKeyCreds struct {
	Key        string
	Secret     string
	Passphrase string
}
