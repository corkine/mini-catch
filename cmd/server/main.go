package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini-catch/internal/config"
	handlers "mini-catch/internal/controller"
	"mini-catch/internal/database"
	"mini-catch/internal/slack"

	"git.mazhangjing.com/corkine/cls-client/data"
)

// App 应用结构
type App struct {
	config   config.Config
	db       *database.Database
	handler  *handlers.Handler
	notifier *slack.Notifier
	server   *http.Server
}

const (
	DATABASE_PATH = "data/mini-catch.db"
)

var Version = "dev"

func main() {
	// 加载配置
	config, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	var svc *data.CLSDataService
	if config.CLS.ProjectURL != "" && config.CLS.ProjectToken != "" {
		svc = data.NewCLSDataService(
			config.CLS.ProjectURL,
			config.CLS.ProjectToken,
			DATABASE_PATH)
	}

	// 如果数据库文件不存在，执行初始化
	if svc != nil {
		if _, err := os.Stat(DATABASE_PATH); os.IsNotExist(err) {
			err := svc.DownloadLatestDB()
			if err != nil {
				log.Fatalf("下载数据失败: %v", err)
			}
			log.Println("✅ 数据已从服务器下载")
		} else {
			log.Println("✅ 数据库文件已存在，跳过下载")
		}
	}

	// 初始化数据库
	db, err := database.NewDatabase(DATABASE_PATH)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化 Slack 通知器
	notifier := &slack.Notifier{Db: db}

	// 初始化处理器
	handler := handlers.NewHandler(db, *config, notifier)

	router := handlers.SetupRoutes(config, handler)

	app := &App{
		config:   *config,
		db:       db,
		handler:  handler,
		notifier: notifier,
		server: &http.Server{
			Addr:         ":" + config.Port,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	log.Printf("🚀 启动 mini-catch 服务器，端口: %s", config.Port)
	log.Printf("📦 版本: %s", Version)

	// 优雅关闭
	go func() {
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭错误: %v", err)
	}

	// 关闭数据库连接
	if err := app.db.Close(); err != nil {
		log.Printf("关闭数据库连接错误: %v", err)
	}

	if svc != nil {
		svc.UploadDB("Upload by MiniCatch " + Version)
		log.Println("✅ 数据已备份到服务器")
	}

	log.Println("✅ 服务器已关闭")
}
