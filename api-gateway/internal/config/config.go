package config

import (
	"log"
	"os"

	"mygoproject/pkg/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DB                      config.DBConfig     `yaml:"db"`
	JWT                     config.JWTConfig    `yaml:"jwt"`
	Server                  config.ServerConfig `yaml:"server"`
	MQ                      config.MQConfig     `yaml:"mq"`
	MailIngestionServiceURL string              `yaml:"mail_ingestion_service_url"`
	TaskServiceURL          string              `yaml:"task_service_url"`
	AgentServiceURL         string              `yaml:"agent_service_url"`
	NotificationServiceURL  string              `yaml:"notification_service_url"`
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
	config.OverrideMQFromEnv(&cfg.MQ)
	if url := os.Getenv("MAIL_INGESTION_SERVICE_URL"); url != "" {
		cfg.MailIngestionServiceURL = url
	}
	if url := os.Getenv("TASK_SERVICE_URL"); url != "" {
		cfg.TaskServiceURL = url
	}
	if url := os.Getenv("AGENT_SERVICE_URL"); url != "" {
		cfg.AgentServiceURL = url
	}
	if url := os.Getenv("NOTIFICATION_SERVICE_URL"); url != "" {
		cfg.NotificationServiceURL = url
	}
	return &cfg
}
