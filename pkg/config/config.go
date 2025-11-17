package config

import (
	"os"
	"strconv"
)

// DBConfig 数据库配置
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// MQConfig 消息队列配置
type MQConfig struct {
	URL string `yaml:"url"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret string `yaml:"secret"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
}

// OverrideDBFromEnv 从环境变量覆盖数据库配置
func OverrideDBFromEnv(cfg *DBConfig) {
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.Password = password
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		cfg.Name = name
	}
}

// OverrideMQFromEnv 从环境变量覆盖MQ配置
func OverrideMQFromEnv(cfg *MQConfig) {
	if url := os.Getenv("MQ_URL"); url != "" {
		cfg.URL = url
	}
}

// OverrideRedisFromEnv 从环境变量覆盖Redis配置
func OverrideRedisFromEnv(cfg *RedisConfig) {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		cfg.Addr = addr
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		cfg.Password = password
	}
}

// OverrideJWTFromEnv 从环境变量覆盖JWT配置
func OverrideJWTFromEnv(cfg *JWTConfig) {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Secret = secret
	}
}

// OverrideServerFromEnv 从环境变量覆盖服务器配置
func OverrideServerFromEnv(cfg *ServerConfig) {
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Port = port
	}
}

