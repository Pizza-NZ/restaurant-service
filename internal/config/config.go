package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server Server `yaml:"server"`

	Database Database `yaml:"database"`

	JWT JWT `yaml:"jwt"`
}

type Server struct {
	Address string `yaml:"address"`
	Mode    string `yaml:"address"`
}

type JWT struct {
	Secret    string `yaml:"secret"`
	ExpiresIn int    `yaml:"expires_in"` // In Hours
}

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func Load() (*Config, error) {
	configPath := "configs/development.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
