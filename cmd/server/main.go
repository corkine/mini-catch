package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini-catch/internal/auth"
	"mini-catch/internal/database"
	"mini-catch/internal/handlers"
	"mini-catch/internal/slack"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
}

// App 应用结构
type App struct {
	config   Config
	db       *database.Database
	handler  *handlers.Handler
	notifier *slack.Notifier
	server   *http.Server
}

// NewApp 创建新的应用
func NewApp(configPath string) (*App, error) {
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

	// 初始化 Slack 通知器
	notifier := slack.NewNotifier(config.SlackWebhookURL)

	// 初始化认证配置
	authConfig := auth.Config{
		Username: config.Auth.Username,
		Password: config.Auth.Password,
	}

	// 初始化处理器
	handler := handlers.NewHandler(db, authConfig, notifier)

	return &App{
		config:   *config,
		db:       db,
		handler:  handler,
		notifier: notifier,
	}, nil
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

// setupRoutes 设置路由
func (a *App) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// 中间件
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.GetHead)

	// 认证中间件
	r.Use(func(next http.Handler) http.Handler {
		return auth.AuthMiddleware(auth.Config{
			Username: a.config.Auth.Username,
			Password: a.config.Auth.Password,
		}, next)
	})

	// API 路由
	r.Route("/api", func(r chi.Router) {
		// 登录接口（不需要认证）
		r.Post("/login", a.handler.LoginHandler)

		// 剧集管理接口
		r.Get("/series", a.handler.GetSeriesList)
		r.Post("/series", a.handler.CreateSeries)
		r.Put("/series/{id}", a.handler.UpdateSeries)
		r.Delete("/series/{id}", a.handler.DeleteSeries)
		r.Post("/series/{id}/watch", a.handler.MarkAsWatched)
		r.Post("/series/{id}/unwatch", a.handler.MarkAsUnwatched)
		r.Post("/series/{id}/toggle-tracking", a.handler.ToggleTracking)
		r.Post("/series/{id}/clear-history", a.handler.ClearSeriesHistory)

		// 爬虫接口
		r.Route("/fetch", func(r chi.Router) {
			r.Get("/", a.handler.HandleFetchTask)
			r.Post("/", a.handler.HandleFetchTaskCallback)
		})
	})

	// 静态文件服务
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/*", fileServer)

	return r
}

var Version = "dev"

// Start 启动应用
func (a *App) Start() error {
	// 设置路由
	r := a.setupRoutes()

	// 创建服务器
	a.server = &http.Server{
		Addr:         ":" + a.config.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 启动 mini-catch 服务器，端口: %s", a.config.Port)
	log.Printf("📦 版本: %s", Version)
	log.Printf("📊 数据库路径: %s", a.config.DatabasePath)
	log.Printf("👤 认证用户: %s", a.config.Auth.Username)
	if a.config.SlackWebhookURL != "" {
		log.Printf("📢 Slack 通知已启用")
	} else {
		log.Printf("📢 Slack 通知未配置")
	}

	// 优雅关闭
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭错误: %v", err)
	}

	// 关闭数据库连接
	if err := a.db.Close(); err != nil {
		log.Printf("关闭数据库连接错误: %v", err)
	}

	log.Println("✅ 服务器已关闭")
	return nil
}

// Close 关闭应用
func (a *App) Close() error {
	if a.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return a.server.Shutdown(ctx)
	}
	return nil
}

// main 主函数
func main() {
	// 创建应用
	application, err := NewApp("config.json")
	if err != nil {
		log.Fatalf("创建应用失败: %v", err)
	}

	// 启动应用
	if err := application.Start(); err != nil {
		log.Fatalf("应用启动失败: %v", err)
	}
}
