package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 应用配置
type Config struct {
	Port string `json:"port"`
	Auth struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"auth"`
	CLS struct {
		PublicKey    string `json:"public_key"`
		MatchPurpose string `json:"match_purpose"`
		RemoteServer string `json:"remote_server"`
		ProjectURL   string `json:"project_url"`
		ProjectToken string `json:"project_token"`
	} `json:"cls"`
}

// loadConfig 加载配置
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadConfig(configPath string) (*Config, error) {
	// 加载配置
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %v", err)
	}

	// 支持环境变量覆盖
	if envUser := os.Getenv("AUTH_USER"); envUser != "" {
		config.Auth.Username = envUser
	}
	if envPass := os.Getenv("AUTH_PASSWORD"); envPass != "" {
		config.Auth.Password = envPass
	}
	if envPort := os.Getenv("MINI_CATCH_PORT"); envPort != "" {
		config.Port = envPort
	}

	return config, nil
}
