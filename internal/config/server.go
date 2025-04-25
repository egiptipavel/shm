package config

type ServerConfig struct {
	Address string
	CommonConfig
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		Address:      getEnv("SERVER_ADDRESS", "server:8080"),
		CommonConfig: NewCommonConfig(),
	}
}
