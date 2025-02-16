package rpc

type RpcConfig struct {
	Port                uint64           `mapstructure:"port"`
	HealthcheckEndpoint string           `mapstructure:"healthcheck_endpoint"`
	HTTP                *HttpConfig      `mapstructure:"http"`
	Websocket           *WebsocketConfig `mapstructure:"websocket"`
}

type HttpConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type WebsocketConfig struct {
	Enabled bool `mapstructure:"enabled"`
}
