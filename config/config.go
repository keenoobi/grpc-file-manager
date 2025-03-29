package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port    string        `mapstructure:"port"`
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"server"`

	Limits struct {
		Upload int `mapstructure:"upload"`
		List   int `mapstructure:"list"`
	} `mapstructure:"limits"`

	Storage struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"storage"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)

	// Устанавливаем значения по умолчанию
	viper.SetDefault("server.port", ":50051")
	viper.SetDefault("limits.upload", 10)
	viper.SetDefault("limits.list", 100)
	viper.SetDefault("storage.path", "./storage")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
