package config

import (
	"log"
	"os"

	"mygoproject/pkg/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DB          config.DBConfig `yaml:"db"`
	MQ          config.MQConfig `yaml:"mq"`
	AgentService struct {
		URL string `yaml:"url"`
	} `yaml:"agent_service"`
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

	if agentURL := os.Getenv("AGENT_SERVICE_URL"); agentURL != "" {
		cfg.AgentService.URL = agentURL
	}

	return &cfg
}

