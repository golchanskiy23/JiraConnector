package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Database      *Database           `mapstructure:"database"`
	ServiceConfig *MicroserviceConfig `mapstructure:"service_config"`
}

type Database struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	DBName   string `mapstructure:"db_name"`
}

type MicroserviceConfig struct {
	Address       string `mapstructure:"address"`
	Port          int    `mapstructure:"port"`
	JiraURL       string `mapstructure:"jiraUrl"`
	Thread        int    `mapstructure:"thread"`
	IssueInOneReq int    `mapstructure:"issueInOneRequest"`
	MaxTimeSleep  string `mapstructure:"maxTimeSleep"`
	MinTimeSleep  string `mapstructure:"minTimeSleep"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("../config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fatal error config file: %s", err)
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("marshaling error: %s", err)
	}
	return cfg, nil
}
