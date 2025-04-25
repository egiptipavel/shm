package config

type SQLiteConfig struct {
	File string
}

func NewSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		File: getEnv("SQLITE_FILE", "storage/shm.db"),
	}
}
