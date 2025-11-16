package config

import (
	"log"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type MQConfig struct {
	URL     string   `yaml:"url"`
	Brokers []string `yaml:"brokers"` // Kafka 用的
	GroupID string   `yaml:"group_id"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type Config struct {
	DB     DBConfig     `yaml:"db"`
	MQ     MQConfig     `yaml:"mq"`
	JWT    JWTConfig    `yaml:"jwt"`
	Server ServerConfig `yaml:"server"`
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

	// 环境变量覆盖（生产环境使用）
	overrideFromEnv(&cfg)

	return &cfg
}

func overrideFromEnv(cfg *Config) {
	// DB配置
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.DB.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.DB.Port = p
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.DB.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.DB.Password = password
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		cfg.DB.Name = name
	}

	// MQ配置
	if url := os.Getenv("MQ_URL"); url != "" {
		cfg.MQ.URL = url
	}

	// JWT配置
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}

	// Server配置
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Server.Port = port
	}
}
