package headers

type L2HeaderArgs struct {
	Method      string
	RequestPath string
	Body        *string
}

type RelayerKey struct {
	ApiKey        string `mapstructure:"key" json:"key"`
	ApiKeyAddress string `mapstructure:"key_address" json:"key_address"`
}
