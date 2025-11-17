package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"mygoproject/pkg/config"
)

type Config struct {
	DB                      config.DBConfig  `yaml:"db"`
	JWT                     config.JWTConfig `yaml:"jwt"`
	Server                  config.ServerConfig `yaml:"server"`
	MailIngestionServiceURL string           `yaml:"mail_ingestion_service_url"`
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

	// 环境变量覆盖
	config.OverrideDBFromEnv(&cfg.DB)
	config.OverrideJWTFromEnv(&cfg.JWT)
	config.OverrideServerFromEnv(&cfg.Server)
	if url := os.Getenv("MAIL_INGESTION_SERVICE_URL"); url != "" {
		cfg.MailIngestionServiceURL = url
	}

	return &cfg
}
