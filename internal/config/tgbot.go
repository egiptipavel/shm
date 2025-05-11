package config

type TelegramBotConfig struct {
	Token string
	CommonConfig
}

func NewTelegramBotConfig() TelegramBotConfig {
	return TelegramBotConfig{
		Token:        getEnvFromFile("TELEGRAM_TOKEN_FILE", getEnv("TELEGRAM_TOKEN", "")),
		CommonConfig: NewCommonConfig(),
	}
}
