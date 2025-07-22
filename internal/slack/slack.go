package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// SlackMessage Slack 消息结构
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment Slack 附件结构
type SlackAttachment struct {
	Color     string  `json:"color,omitempty"`
	Title     string  `json:"title,omitempty"`
	TitleLink string  `json:"title_link,omitempty"`
	Text      string  `json:"text,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
	Footer    string  `json:"footer,omitempty"`
	Ts        int64   `json:"ts,omitempty"`
}

// Field Slack 字段结构
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Notifier Slack 通知器
type Notifier struct {
	webhookURL string
}

// NewNotifier 创建新的 Slack 通知器
func NewNotifier(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
	}
}

// SendNotification 发送 Slack 通知
func (n *Notifier) SendNotification(seriesName string, newEpisodes []string, url string) {
	if n.webhookURL == "" {
		log.Printf("Slack Webhook URL 未配置，跳过通知")
		return
	}

	// 构建消息
	message := n.buildSlackMessage(seriesName, newEpisodes, url)

	// 发送到 Slack
	if err := n.sendToSlack(message); err != nil {
		log.Printf("发送 Slack 通知失败: %v", err)
	} else {
		log.Printf("已发送 Slack 通知: %s 新增 %d 集", seriesName, len(newEpisodes))
	}
}

// 构建 Slack 消息
func (n *Notifier) buildSlackMessage(seriesName string, newEpisodes []string, url string) SlackMessage {
	// 格式化新集数列表
	episodeList := strings.Join(newEpisodes, ", ")

	// 构建附件
	attachment := SlackAttachment{
		Color:     "#36a64f", // 绿色
		Title:     fmt.Sprintf("🎬 %s 有新集数更新！", seriesName),
		TitleLink: url,
		Text:      fmt.Sprintf("发现 %d 集新内容", len(newEpisodes)),
		Fields: []Field{
			{
				Title: "剧集名称",
				Value: seriesName,
				Short: true,
			},
			{
				Title: "新增集数",
				Value: episodeList,
				Short: false,
			},
		},
		Footer: "mini-catch 自动追踪",
		Ts:     time.Now().Unix(),
	}

	return SlackMessage{
		Attachments: []SlackAttachment{attachment},
	}
}

// 发送到 Slack
func (n *Notifier) sendToSlack(message SlackMessage) error {
	// 序列化消息
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 发送 HTTP 请求
	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API 返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// SendTestNotification 发送测试通知
func (n *Notifier) SendTestNotification() error {
	if n.webhookURL == "" {
		return fmt.Errorf("Slack Webhook URL 未配置")
	}

	message := SlackMessage{
		Text: "🧪 mini-catch 测试通知\n如果您看到这条消息，说明 Slack 通知功能配置正确！",
	}

	return n.sendToSlack(message)
}

// SendUpdateStatusNotification 发送剧集更新状态变更的 Slack 通知
func (n *Notifier) SendUpdateStatusNotification(seriesName, oldStatus, newStatus, url string) {
	if n.webhookURL == "" {
		log.Printf("Slack Webhook URL 未配置，跳过通知")
		return
	}

	attachment := SlackAttachment{
		Color:     "#439FE0", // 蓝色
		Title:     fmt.Sprintf("🎬 %s 更新状态变更", seriesName),
		TitleLink: url,
		Text:      fmt.Sprintf("更新状态: %s → %s", oldStatus, newStatus),
		Fields: []Field{
			{
				Title: "剧集名称",
				Value: seriesName,
				Short: true,
			},
			{
				Title: "原状态",
				Value: oldStatus,
				Short: true,
			},
			{
				Title: "新状态",
				Value: newStatus,
				Short: true,
			},
		},
		Footer: "mini-catch 自动追踪",
		Ts:     time.Now().Unix(),
	}

	message := SlackMessage{
		Attachments: []SlackAttachment{attachment},
	}

	if err := n.sendToSlack(message); err != nil {
		log.Printf("发送 Slack 更新状态通知失败: %v", err)
	} else {
		log.Printf("已发送 Slack 更新状态通知: %s %s → %s", seriesName, oldStatus, newStatus)
	}
}
