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
	// 使用统一配置中心
	env := config.GetConfigEnv()
	configDir := config.GetEnv("CONFIG_DIR", "config")

	cfgMap, err := config.LoadConfig(env, configDir)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 转换为 Config 结构
	var cfg Config
	cfgData, err := yaml.Marshal(cfgMap)
	if err != nil {
		log.Fatalf("failed to marshal config: %v", err)
	}
	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	// 从 services 配置中提取服务 URL
	if services, ok := cfgMap["services"].(map[string]interface{}); ok {
		if url, ok := services["mail_ingestion"].(string); ok {
			cfg.MailIngestionServiceURL = url
		}
		if url, ok := services["task"].(string); ok {
			cfg.TaskServiceURL = url
		}
		if url, ok := services["agent"].(string); ok {
			cfg.AgentServiceURL = url
		}
		if url, ok := services["notification"].(string); ok {
			cfg.NotificationServiceURL = url
		}
	}

	// 环境变量覆盖（优先级最高）
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
