package config

import (
	"log"
	"os"

	"mygoproject/pkg/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DB          config.DBConfig `yaml:"db"`
	MQ          config.MQConfig  `yaml:"mq"`
	Notification struct {
		RetryMax         int `yaml:"retry_max"`
		RetryDelaySeconds int `yaml:"retry_delay_seconds"`
	} `yaml:"notification"`
}

func Load() *Config {
	f, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("failed to open config.yaml: %v", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("failed to decode config.yaml: %v", err)
	}

	config.OverrideDBFromEnv(&cfg.DB)
	config.OverrideMQFromEnv(&cfg.MQ)

	return &cfg
}

