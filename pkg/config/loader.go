package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig 加载配置，支持多环境
// env: local, production, 或其他环境名称
// configDir: 配置文件目录，默认为 "config"
func LoadConfig(env string, configDir string) (map[string]interface{}, error) {
	if configDir == "" {
		configDir = "config"
	}

	// 1. 加载 base.yaml
	baseConfig, err := loadYAMLFile(filepath.Join(configDir, "base.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to load base.yaml: %w", err)
	}

	// 2. 加载环境特定配置（如果存在）
	envConfig := make(map[string]interface{})
	if env != "" && env != "base" {
		envFile := filepath.Join(configDir, fmt.Sprintf("%s.yaml", env))
		if _, err := os.Stat(envFile); err == nil {
			envConfig, err = loadYAMLFile(envFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load %s.yaml: %w", env, err)
			}
		}
	}

	// 3. 合并配置（环境配置覆盖基础配置）
	merged := mergeMaps(baseConfig, envConfig)

	// 4. 加载 secrets.env（如果存在）
	secretsFile := filepath.Join(configDir, "secrets.env")
	if _, err := os.Stat(secretsFile); err == nil {
		secrets, err := loadEnvFile(secretsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load secrets.env: %w", err)
		}
		// 用环境变量覆盖配置中的占位符
		merged = substituteEnvVars(merged, secrets)
	}

	// 5. 用系统环境变量覆盖（优先级最高）
	merged = overrideFromSystemEnv(merged)

	return merged, nil
}

// loadYAMLFile 加载 YAML 文件
func loadYAMLFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// loadEnvFile 加载 .env 文件
func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	env := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// 移除引号
			value = strings.Trim(value, `"`)
			value = strings.Trim(value, `'`)
			env[key] = value
		}
	}

	return env, nil
}

// mergeMaps 合并两个 map，dst 会被 src 覆盖
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// 复制 dst
	for k, v := range dst {
		result[k] = v
	}

	// 覆盖/合并 src
	for k, v := range src {
		if dstMap, ok := result[k].(map[string]interface{}); ok {
			if srcMap, ok := v.(map[string]interface{}); ok {
				// 递归合并嵌套 map
				result[k] = mergeMaps(dstMap, srcMap)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

// substituteEnvVars 替换配置中的环境变量占位符 ${VAR_NAME}
func substituteEnvVars(config map[string]interface{}, env map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range config {
		switch val := v.(type) {
		case string:
			result[k] = substituteString(val, env)
		case map[string]interface{}:
			result[k] = substituteEnvVars(val, env)
		default:
			result[k] = v
		}
	}
	return result
}

// substituteString 替换字符串中的环境变量
func substituteString(s string, env map[string]string) string {
	if !strings.Contains(s, "${") {
		return s
	}

	result := s
	for key, value := range env {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// overrideFromSystemEnv 用系统环境变量覆盖配置
func overrideFromSystemEnv(config map[string]interface{}) map[string]interface{} {
	// 这里可以添加特定的环境变量覆盖逻辑
	// 目前依赖现有的 Override*FromEnv 函数
	return config
}

// GetEnv 获取环境变量，如果未设置则返回默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetConfigEnv 获取配置环境（从环境变量 CONFIG_ENV，默认为 local）
func GetConfigEnv() string {
	return GetEnv("CONFIG_ENV", "local")
}

