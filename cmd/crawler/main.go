package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"mini-catch/internal/crawler"
)

var Version = "dev"

func main() {
	// 命令行参数
	var (
		serverURL = flag.String("server", "", "服务器 URL (必需)")
		username  = flag.String("username", "", "用户名 (必需)")
		password  = flag.String("password", "", "密码 (必需)")
		debug     = flag.Bool("debug", false, "调试模式")
		headless  = flag.Bool("headless", true, "无头模式")
		timeout   = flag.Int("timeout", 120, "超时时间（秒）")
	)
	flag.Parse()

	// 如果没有命令行参数，则从环境变量读取
	if *serverURL == "" {
		*serverURL = os.Getenv("SERVER_URL")
	}
	if *username == "" {
		*username = os.Getenv("USERNAME")
	}
	if *password == "" {
		*password = os.Getenv("PASSWORD")
	}
	if os.Getenv("TIMEOUT") != "" {
		to, err := strconv.Atoi(os.Getenv("TIMEOUT"))
		if err != nil {
			log.Fatal("❌ 超时时间格式错误")
		}
		*timeout = to
	}

	// 检查必需参数
	if *serverURL == "" || *username == "" || *password == "" {
		log.Fatal("❌ 缺少必需参数: --server, --username, --password，或环境变量 SERVER_URL、USERNAME、PASSWORD")
	}

	log.Printf("🐛 mini-catch-crawler 版本: %s", Version)
	log.Println("🚀 启动 mini4k 爬虫")
	log.Printf("📡 服务器: %s", *serverURL)
	log.Printf("👤 用户名: %s", *username)
	log.Printf("🐛 调试模式: %v", *debug)
	log.Printf("👻 无头模式: %v", *headless)
	log.Printf("⏰ 超时时间: %d 秒", *timeout)

	// 创建配置
	config := &crawler.Config{
		ServerURL: *serverURL,
		Username:  *username,
		Password:  *password,
		Debug:     *debug,
		Headless:  *headless,
		Timeout:   *timeout,
	}

	// 创建爬虫实例
	c := crawler.NewMini4KCrawler(config)

	// 运行爬虫
	if err := c.Run(); err != nil {
		log.Printf("❌ 爬虫运行失败: %v", err)
		os.Exit(1)
	}

	log.Println("🎉 爬虫任务完成")
}
