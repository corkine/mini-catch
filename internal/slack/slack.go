package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mini-catch/internal/database"
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
	Db *database.Database
}

// getWebhookURL 从数据库获取 webhook URL
func (n *Notifier) getWebhookURL() string {
	if n.Db == nil {
		return ""
	}

	settings, err := n.Db.GetSettings()
	if err != nil {
		log.Printf("获取设置失败: %v", err)
		return ""
	}

	return settings.SlackWebhookURL
}

// send 发送消息到配置的 webhook
func (n *Notifier) send(message SlackMessage) error {
	webhookURL := n.getWebhookURL()
	if webhookURL == "" {
		return fmt.Errorf("slack Webhook URL 未配置")
	}

	// 序列化消息
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 发送 HTTP 请求
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API 返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// SendMessage 发送自定义消息
func (n *Notifier) SendMessage(messageText string) error {
	message := SlackMessage{Text: messageText}
	return n.send(message)
}

// SendTestNotification 发送测试通知
func (n *Notifier) SendTestNotification() error {
	message := SlackMessage{
		Text: "🧪 MiniCatch 测试通知\n如果您看到这条消息，说明 Slack 通知功能配置正确！",
	}
	return n.send(message)
}

// SendNotification 发送剧集更新通知
func (n *Notifier) SendNotification(seriesName string, newEpisodes []string, url string) {
	message := n.buildEpisodeUpdateMessage(seriesName, newEpisodes, url)

	if err := n.send(message); err != nil {
		log.Printf("发送 Slack 通知失败: %v", err)
	} else {
		log.Printf("已发送 Slack 通知: %s 新增 %d 集", seriesName, len(newEpisodes))
	}
}

// SendStatusUpdateNotification 发送剧集状态变更通知
func (n *Notifier) SendStatusUpdateNotification(seriesName, oldStatus, newStatus, url string) {
	message := n.buildStatusChangeMessage(seriesName, oldStatus, newStatus, url)

	if err := n.send(message); err != nil {
		log.Printf("发送 Slack 更新状态通知失败: %v", err)
	} else {
		log.Printf("已发送 Slack 更新状态通知: %s %s → %s", seriesName, oldStatus, newStatus)
	}
}

// buildStatusChangeMessage 构建状态变更消息
func (n *Notifier) buildStatusChangeMessage(seriesName, oldStatus, newStatus, url string) SlackMessage {
	attachment := SlackAttachment{
		Color:     "#439FE0", // 蓝色
		Title:     fmt.Sprintf("🎬 %s %s", seriesName, newStatus),
		TitleLink: url,
		Text:      fmt.Sprintf("%s有更新: %s", seriesName, newStatus),
		Fields: []Field{
			{Title: "剧集名称", Value: seriesName, Short: true},
			{Title: "原状态", Value: oldStatus, Short: true},
			{Title: "新状态", Value: newStatus, Short: true},
		},
		Footer: "MiniCatch 自动追踪",
		Ts:     time.Now().Unix(),
	}

	return SlackMessage{Attachments: []SlackAttachment{attachment}}
}

// buildEpisodeUpdateMessage 构建剧集更新消息
func (n *Notifier) buildEpisodeUpdateMessage(seriesName string, newEpisodes []string, url string) SlackMessage {
	episodeList := strings.Join(newEpisodes, ", ")

	attachment := SlackAttachment{
		Color:     "#36a64f", // 绿色
		Title:     fmt.Sprintf("🎬 %s 有新集数更新！", seriesName),
		TitleLink: url,
		Text:      fmt.Sprintf("发现 %d 集新内容", len(newEpisodes)),
		Fields: []Field{
			{Title: "剧集名称", Value: seriesName, Short: true},
			{Title: "新增集数", Value: episodeList, Short: false},
		},
		Footer: "MiniCatch 自动追踪",
		Ts:     time.Now().Unix(),
	}

	return SlackMessage{Attachments: []SlackAttachment{attachment}}
}
