package config

type AlertServiceConfig struct {
	NumberOrFailedChecks int
	CommonConfig
}

func NewAlertServiceConfig() AlertServiceConfig {
	return AlertServiceConfig{
		NumberOrFailedChecks: getEnvAsInt("NUMBER_OF_FAILED_CHECKS", 3),
		CommonConfig:         NewCommonConfig(),
	}
}
