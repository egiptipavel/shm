package config

type MigratorConfig struct {
	MigrationsFolder string
	CommonConfig
}

func NewMigratorConfig() MigratorConfig {
	return MigratorConfig{
		MigrationsFolder: getEnv("MIGRATIONS_FOLDER", "migrations/postgres"),
		CommonConfig:     NewCommonConfig(),
	}
}
