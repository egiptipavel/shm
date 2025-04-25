package config

type AlertServiceConfig struct {
	CommonConfig
}

func NewAlertServiceConfig() AlertServiceConfig {
	return AlertServiceConfig{
		CommonConfig: NewCommonConfig(),
	}
}
