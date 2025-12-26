package headers

type L2HeaderArgs struct {
	Method      string
	RequestPath string
	Body        *string
}

type ApiKeyCreds struct {
	Key        string `json:"key"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}
