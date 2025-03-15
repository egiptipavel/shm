package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Interval int
	DBFile   string
	Token    string
}

func ParseConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := Config{
		DBFile: "shm.db",
	}
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
