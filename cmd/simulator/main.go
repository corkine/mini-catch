package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// 爬虫模拟器配置
type CrawlerConfig struct {
	BaseURL    string `json:"base_url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Interval   int    `json:"interval"`    // 爬取间隔（秒）
	MaxRetries int    `json:"max_retries"` // 最大重试次数
}

// 爬虫模拟器
type CrawlerSimulator struct {
	config     CrawlerConfig
	authToken  string
	httpClient *http.Client
}

// 创建新的爬虫模拟器
func NewCrawlerSimulator(config CrawlerConfig) *CrawlerSimulator {
	return &CrawlerSimulator{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 登录获取认证令牌
func (c *CrawlerSimulator) login() error {
	loginData := map[string]string{
		"username": c.config.Username,
		"password": c.config.Password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("序列化登录数据失败: %v", err)
	}

	resp, err := c.httpClient.Post(
		c.config.BaseURL+"/api/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("登录失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析登录响应失败: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("登录失败: %s", result.Message)
	}

	c.authToken = result.Data.Token
	log.Printf("✅ 登录成功，获取到认证令牌")
	return nil
}

// 获取爬虫任务
func (c *CrawlerSimulator) fetchTasks() (*FetchTask, error) {
	req, err := http.NewRequest("GET", c.config.BaseURL+"/fetch", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取任务失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		Success bool      `json:"success"`
		Data    FetchTask `json:"data"`
		Message string    `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析任务响应失败: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("获取任务失败: %s", result.Message)
	}

	log.Printf("📋 获取到 %d 个爬虫任务", len(result.Data.URLs))
	return &result.Data, nil
}

// 模拟爬取数据
func (c *CrawlerSimulator) crawlData(urls []string) []FetchResult {
	var results []FetchResult

	for _, url := range urls {
		// 模拟爬取延迟
		time.Sleep(500 * time.Millisecond)

		// 模拟不同的爬取结果
		result := c.simulateCrawlResult(url)
		results = append(results, result)

		log.Printf("🕷️ 模拟爬取: %s -> %s", url, result.Name)
	}

	return results
}

// 模拟爬取结果
func (c *CrawlerSimulator) simulateCrawlResult(url string) FetchResult {
	// 根据 URL 生成不同的模拟数据
	episodeCount := len(url)%10 + 1 // 1-10 集
	seasonCount := len(url)%3 + 1   // 1-3 季

	var series []string
	for season := 1; season <= seasonCount; season++ {
		for episode := 1; episode <= episodeCount; episode++ {
			series = append(series, fmt.Sprintf("S%02dE%02d", season, episode))
		}
	}

	// 随机选择一些集数作为"新发现"
	newEpisodes := len(series)%3 + 1
	if newEpisodes > len(series) {
		newEpisodes = len(series)
	}

	// 模拟发现新集数
	discoveredSeries := series[:newEpisodes]
	currentEpisode := discoveredSeries[len(discoveredSeries)-1]

	return FetchResult{
		Name:   fmt.Sprintf("模拟剧集-%s", url[len(url)-8:]), // 取 URL 后8位作为名称
		Update: currentEpisode,
		URL:    url,
		Series: discoveredSeries,
	}
}

// 上报爬取结果
func (c *CrawlerSimulator) reportResults(tasks []string, results []FetchResult) error {
	callbackData := FetchCallback{
		Tasks:   tasks,
		Results: results,
		Status:  0,
		Message: "success",
	}

	jsonData, err := json.Marshal(callbackData)
	if err != nil {
		return fmt.Errorf("序列化回调数据失败: %v", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL+"/fetch", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建回调请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("上报结果失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上报结果失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析回调响应失败: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("上报结果失败: %s", result.Message)
	}

	log.Printf("📤 成功上报 %d 个爬取结果", len(results))
	return nil
}

// 运行爬虫模拟器
func (c *CrawlerSimulator) Run() error {
	log.Printf("🚀 启动爬虫模拟器")
	log.Printf("📡 目标服务器: %s", c.config.BaseURL)
	log.Printf("👤 用户名: %s", c.config.Username)
	log.Printf("⏰ 爬取间隔: %d 秒", c.config.Interval)

	// 首次登录
	if err := c.login(); err != nil {
		return fmt.Errorf("初始登录失败: %v", err)
	}

	// 循环爬取
	for {
		log.Printf("\n🔄 开始新一轮爬取...")

		// 获取任务
		task, err := c.fetchTasks()
		if err != nil {
			log.Printf("❌ 获取任务失败: %v", err)

			// 尝试重新登录
			if err := c.login(); err != nil {
				log.Printf("❌ 重新登录失败: %v", err)
			}

			time.Sleep(time.Duration(c.config.Interval) * time.Second)
			continue
		}

		if len(task.URLs) == 0 {
			log.Printf("📭 没有需要爬取的任务")
			time.Sleep(time.Duration(c.config.Interval) * time.Second)
			continue
		}

		// 模拟爬取数据
		results := c.crawlData(task.URLs)

		// 上报结果
		if err := c.reportResults(task.URLs, results); err != nil {
			log.Printf("❌ 上报结果失败: %v", err)
		}

		log.Printf("✅ 本轮爬取完成，等待 %d 秒后继续...", c.config.Interval)
		time.Sleep(time.Duration(c.config.Interval) * time.Second)
	}
}

// 主函数 - 爬虫模拟器入口
func main() {
	// 配置爬虫模拟器
	config := CrawlerConfig{
		BaseURL:    "http://localhost:8080",
		Username:   "admin",
		Password:   "admin123",
		Interval:   30, // 30秒爬取一次
		MaxRetries: 3,
	}

	// 创建爬虫模拟器
	crawler := NewCrawlerSimulator(config)

	// 运行爬虫
	if err := crawler.Run(); err != nil {
		log.Fatalf("爬虫运行失败: %v", err)
	}
}

// 数据结构定义（与主程序保持一致）
type FetchTask struct {
	URLs        []string `json:"tasks"`
	CallbackURL string   `json:"callback_url"`
}

type FetchResult struct {
	Name   string   `json:"name"`
	Update string   `json:"update"`
	URL    string   `json:"url"`
	Series []string `json:"series"`
}

type FetchCallback struct {
	Tasks   []string      `json:"tasks"`
	Results []FetchResult `json:"results"`
	Status  int           `json:"status"`
	Message string        `json:"message"`
}
