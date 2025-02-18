package config

import (
	"encoding/json"
	"os"
	"shm/internal/utils"
)

type Config struct {
	Sites    []string
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

	config.Sites = utils.Distinct(config.Sites)

	return &config, nil
}
