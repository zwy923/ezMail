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

	// 从 services 配置中提取 agent URL
	if services, ok := cfgMap["services"].(map[string]interface{}); ok {
		if url, ok := services["agent"].(string); ok {
			cfg.AgentService.URL = url
		}
	}

	// 环境变量覆盖（优先级最高）
	config.OverrideDBFromEnv(&cfg.DB)
	config.OverrideMQFromEnv(&cfg.MQ)
	if agentURL := os.Getenv("AGENT_SERVICE_URL"); agentURL != "" {
		cfg.AgentService.URL = agentURL
	}

	return &cfg
}

