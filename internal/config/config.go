package config

import (
	"encoding/json"
	"fmt"
	"mini-catch/internal/database"
	"os"
)

// Config 应用配置
type Config struct {
	Port            string `json:"port"`
	DatabasePath    string `json:"database_path"`
	SlackWebhookURL string `json:"slack_webhook_url"`
	Auth            struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"auth"`
	CLS struct {
		PublicKey    string `json:"public_key"`
		MatchPurpose string `json:"match_purpose"`
		RemoteServer string `json:"remote_server"`
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

	// 创建数据目录
	if err := os.MkdirAll("data", 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 初始化数据库
	db, err := database.NewDatabase(config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %v", err)
	}

	if err := db.CreateTables(); err != nil {
		return nil, fmt.Errorf("创建数据库表失败: %v", err)
	}

	return config, nil
}
