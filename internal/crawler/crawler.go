package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mini-catch/internal/database"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/chromedp/chromedp"
)

// Config 爬虫配置
type Config struct {
	ServerURL string `json:"server_url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Debug     bool   `json:"debug"`
	Headless  bool   `json:"headless"`
	Timeout   int    `json:"timeout"`
}

// SeriesInfo 剧集信息
type SeriesInfo struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Update string   `json:"update"`
	Series []string `json:"series"`
}

// CrawlResult 爬取结果
type CrawlResult struct {
	Tasks   []string     `json:"tasks"`
	Results []SeriesInfo `json:"results"`
	Status  int          `json:"status"`
	Message string       `json:"message"`
}

// Mini4KCrawler mini4k 爬虫
type Mini4KCrawler struct {
	config     *Config
	authToken  string
	httpClient *http.Client
}

// NewMini4KCrawler 创建新的爬虫实例
func NewMini4KCrawler(config *Config) *Mini4KCrawler {
	return &Mini4KCrawler{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// login 登录获取认证令牌
func (c *Mini4KCrawler) login() error {
	loginData := map[string]string{
		"username": c.config.Username,
		"password": c.config.Password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("序列化登录数据失败: %v", err)
	}

	resp, err := c.httpClient.Post(
		c.config.ServerURL+"/api/login",
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

// fetchTasks 获取爬虫任务
func (c *Mini4KCrawler) fetchTasks() (*database.FetchTask, error) {
	req, err := http.NewRequest("GET", c.config.ServerURL+"/api/fetch", nil)
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
		Success bool               `json:"success"`
		Data    database.FetchTask `json:"data"`
		Message string             `json:"message"`
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

// printAgent 打印代理信息
func (c *Mini4KCrawler) printAgent(content string) {
	log.Println(content)
}

// getChromeOptions 获取 Chrome 选项
func (c *Mini4KCrawler) getChromeOptions() []chromedp.ExecAllocatorOption {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "SameSiteByDefaultCookies,CookiesWithoutSameSiteMustBeSecure"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-images", false),
		chromedp.Flag("disable-javascript", false),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-field-trial-config", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-component-extensions-with-background-pages", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
	}

	if c.config.Headless {
		opts = append(opts, chromedp.Flag("headless", true))
	}

	return opts
}

// fetchMini4KSeries 爬取 mini4k 剧集信息
func (c *Mini4KCrawler) fetchMini4KSeries(url string) (*SeriesInfo, error) {
	c.printAgent("launching driver...")

	// 创建 Chrome 选项
	opts := c.getChromeOptions()

	// 创建 Chrome 实例
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 创建新的 Chrome 标签页
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
	defer cancel()

	var seriesInfo SeriesInfo

	// 执行爬取任务
	err := chromedp.Run(ctx, chromedp.Tasks{
		// 注入 JavaScript 来隐藏 webdriver 属性
		chromedp.Evaluate(`
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});
		`, nil),

		// 访问页面
		chromedp.Navigate(url),

		// 等待页面加载
		chromedp.Sleep(3 * time.Second),

		// 获取剧集名称
		chromedp.Text(".ch-title", &seriesInfo.Name, chromedp.ByQuery),

		// 获取更新状态（如果存在）
		chromedp.ActionFunc(func(ctx context.Context) error {
			var exists bool
			err := chromedp.Evaluate(`document.querySelector(".tv-status.runing") !== null`, &exists).Do(ctx)
			if err != nil {
				return err
			}
			if exists {
				return chromedp.Text(".tv-status.runing", &seriesInfo.Update, chromedp.ByQuery).Do(ctx)
			}
			// 不存在则在后续处理
			seriesInfo.Update = ""
			return nil
		}),

		// 获取所有剧集链接
		chromedp.Evaluate(`
			(() => {
				const links = document.querySelectorAll('td a[hreflang="zh-hans"]');
				return Array.from(links).map(link => link.title);
			})()
		`, &seriesInfo.Series),
	})

	if err != nil {
		return nil, fmt.Errorf("爬取页面失败: %v", err)
	}

	// 处理剧集信息
	seriesInfo.URL = url

	// 从剧集标题中提取季集信息
	var episodes []string
	episodeRegex := regexp.MustCompile(`(S\d+E\d+)`)

	for _, series := range seriesInfo.Series {
		if matches := episodeRegex.FindStringSubmatch(series); len(matches) > 1 {
			episodes = append(episodes, matches[1])
		}
	}

	// 去重并排序
	episodeMap := make(map[string]bool)
	for _, episode := range episodes {
		episodeMap[episode] = true
	}

	seriesInfo.Series = make([]string, 0, len(episodeMap))
	for episode := range episodeMap {
		seriesInfo.Series = append(seriesInfo.Series, episode)
	}
	sort.Strings(seriesInfo.Series)

	// 如果 Update 为空，取 Series 最后一个
	if seriesInfo.Update == "" && len(seriesInfo.Series) > 0 {
		seriesInfo.Update = fmt.Sprintf("已更新到 %s", seriesInfo.Series[len(seriesInfo.Series)-1])
	}

	c.printAgent("quiting driver...")

	return &seriesInfo, nil
}

// fetchMini4KSeriesWithRetry 带重试的爬取
func (c *Mini4KCrawler) fetchMini4KSeriesWithRetry(url string) (*SeriesInfo, error) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		c.printAgent(fmt.Sprintf("fetching url %s (attempt %d/%d)", url, i+1, maxRetries))

		result, err := c.fetchMini4KSeries(url)
		if err == nil {
			return result, nil
		}

		c.printAgent(fmt.Sprintf("attempt %d failed: %v", i+1, err))
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	return nil, fmt.Errorf("爬取失败，已重试 %d 次", maxRetries)
}

// crawlTasks 爬取任务列表
func (c *Mini4KCrawler) crawlTasks(urls []string) []SeriesInfo {
	var results []SeriesInfo

	for _, url := range urls {
		result, err := c.fetchMini4KSeriesWithRetry(url)
		if err != nil {
			log.Printf("爬取 %s 失败: %v", url, err)
			continue
		}

		results = append(results, *result)
		log.Printf("成功爬取: %s -> %s", url, result.Name)
	}

	return results
}

// reportResults 上报爬取结果
func (c *Mini4KCrawler) reportResults(tasks []string, results []SeriesInfo) error {
	callbackData := CrawlResult{
		Tasks:   tasks,
		Results: results,
		Status:  1,
		Message: "success",
	}

	jsonData, err := json.Marshal(callbackData)
	if err != nil {
		return fmt.Errorf("序列化回调数据失败: %v", err)
	}

	req, err := http.NewRequest("POST", c.config.ServerURL+"/api/fetch", bytes.NewBuffer(jsonData))
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

	log.Printf("📤 上报结果: %#v", callbackData)
	return nil
}

// Run 运行爬虫
func (c *Mini4KCrawler) Run() error {
	log.Printf("🚀 启动 mini4k 爬虫")
	log.Printf("📡 目标服务器: %s", c.config.ServerURL)
	log.Printf("👤 用户名: %s", c.config.Username)

	// 登录获取认证令牌
	if err := c.login(); err != nil {
		return fmt.Errorf("登录失败: %v", err)
	}

	// 获取任务
	task, err := c.fetchTasks()
	if err != nil {
		return fmt.Errorf("获取任务失败: %v", err)
	}

	if len(task.URLs) == 0 {
		log.Printf("📭 没有需要爬取的任务")
		return nil
	}

	// 爬取任务
	log.Printf("🕷️ 开始爬取 %d 个任务", len(task.URLs))
	results := c.crawlTasks(task.URLs)

	// 上报结果
	if err := c.reportResults(task.URLs, results); err != nil {
		return fmt.Errorf("上报结果失败: %v", err)
	}

	log.Printf("✅ 爬虫任务完成，成功爬取 %d 个结果", len(results))
	return nil
}
