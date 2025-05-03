package config

type PostgreSQLConfig struct {
	User string
	Pass string
	Host string
	Port string
	Db   string
}

func NewPostgreSQLConfig() PostgreSQLConfig {
	return PostgreSQLConfig{
		User: getEnv("POSTGRES_USER", "postgres"),
		Pass: getEnvFromFile("POSTGRES_PASSWORD_FILE", "postgres"),
		Host: getEnv("POSTGRES_IP_ADDRESS", "postgres"),
		Port: getEnv("POSTGRES_PORT", "5432"),
		Db:   getEnv("POSTGRES_DB", "postgres_db"),
	}
}
