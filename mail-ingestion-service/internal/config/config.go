package config

import (
	"log"

	"gopkg.in/yaml.v3"
	"mygoproject/pkg/config"
)

type Config struct {
	DB     config.DBConfig     `yaml:"db"`
	MQ     config.MQConfig     `yaml:"mq"`
	Server config.ServerConfig `yaml:"server"`
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

	// 环境变量覆盖（优先级最高）
	config.OverrideDBFromEnv(&cfg.DB)
	config.OverrideMQFromEnv(&cfg.MQ)
	config.OverrideServerFromEnv(&cfg.Server)

	return &cfg
}
