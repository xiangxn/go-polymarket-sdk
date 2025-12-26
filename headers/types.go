package headers

type L2HeaderArgs struct {
	Method      string
	RequestPath string
	Body        *string
}

type ApiKeyCreds struct {
	Key        string `mapstructure:"key" json:"key"`
	Secret     string `mapstructure:"secret" json:"secret"`
	Passphrase string `mapstructure:"passphrase" json:"passphrase"`
}
