package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"mygoproject/pkg/config"
)

type Config struct {
	DB     config.DBConfig     `yaml:"db"`
	MQ     config.MQConfig     `yaml:"mq"`
	Server config.ServerConfig `yaml:"server"`
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
	config.OverrideServerFromEnv(&cfg.Server)

	return &cfg
}
