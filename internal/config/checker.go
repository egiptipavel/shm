package config

type CheckerConfig struct {
	Workers int
	CommonConfig
}

func NewCheckerConfig() CheckerConfig {
	return CheckerConfig{
		Workers:      getEnvAsInt("CHECKER_WORKERS", 1000),
		CommonConfig: NewCommonConfig(),
	}
}
