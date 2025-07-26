package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"mini-catch/internal/config"
	"mini-catch/internal/database"
	"mini-catch/internal/slack"

	"git.mazhangjing.com/corkine/cls-client/auth"

	"github.com/go-chi/chi/v5"
)

// Handler HTTP处理器
type Handler struct {
	db       *database.Database
	config   config.Config
	notifier *slack.Notifier
	cls      *auth.CLSAuthService
}

// NewHandler 创建新的处理器
func NewHandler(db *database.Database, config config.Config, notifier *slack.Notifier) *Handler {
	var clsSvc *auth.CLSAuthService
	if config.CLS.PublicKey != "" && config.CLS.MatchPurpose != "" && config.CLS.RemoteServer != "" {
		clsSvc = auth.NewCLSAuthService(config.CLS.PublicKey, config.CLS.MatchPurpose, config.CLS.RemoteServer)
	} else {
		log.Printf("No valid CLS Config found, skip")
	}
	return &Handler{
		db:       db,
		config:   config,
		notifier: notifier,
		cls:      clsSvc,
	}
}

// 响应结构
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 登录响应结构
type LoginResponse struct {
	Token string `json:"token"`
}

// LoginHandler 登录处理器
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 如果用户名是 CLS，则使用 CLS JWT 认证
	// 如果用户名是 CLST，则使用 CLS Token 认证
	switch req.Username {
	case "CLS":
		if h.cls == nil {
			h.errorResponse(w, http.StatusUnauthorized, "CLS 认证未配置")
			return
		}
		claims, err := h.cls.JwtAuth(req.Password)
		if err != nil {
			h.errorResponse(w, http.StatusUnauthorized, "认证失败: "+err.Error())
			return
		}
		log.Printf("CLS JWT 认证成功: %+v", claims)
	case "CLST":
		if h.cls == nil {
			h.errorResponse(w, http.StatusUnauthorized, "CLS 认证未配置")
			return
		}
		claims, err := h.cls.TokenAuth(req.Password)
		if err != nil {
			h.errorResponse(w, http.StatusUnauthorized, "认证失败: "+err.Error())
			return
		}
		log.Printf("CLS Token 认证成功: %+v", claims)
	default:
		// 验证用户名和密码
		if req.Username != h.config.Auth.Username || req.Password != h.config.Auth.Password {
			h.errorResponse(w, http.StatusUnauthorized, "用户名或密码错误")
			return
		}
	}

	// 生成认证令牌
	token := GenerateAuthToken(h.config.Auth.Username, h.config.Auth.Password)

	// 设置 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // 在生产环境中应该设置为 true
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour), // 24小时过期
	})

	h.successResponse(w, LoginResponse{Token: token})
}

// GetSeriesList 获取剧集列表
func (h *Handler) GetSeriesList(w http.ResponseWriter, r *http.Request) {
	series, err := h.db.GetAllSeries()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "获取剧集列表失败: "+err.Error())
		return
	}

	h.successResponse(w, series)
}

// CreateSeries 创建剧集
func (h *Handler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if req.Name == "" || req.URL == "" {
		h.errorResponse(w, http.StatusBadRequest, "名称和URL不能为空")
		return
	}

	series, err := h.db.CreateSeries(req.Name, req.URL)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "创建剧集失败: "+err.Error())
		return
	}

	h.successResponse(w, series)
}

// UpdateSeries 更新剧集
func (h *Handler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "无效的ID")
		return
	}

	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if req.Name == "" || req.URL == "" {
		h.errorResponse(w, http.StatusBadRequest, "名称和URL不能为空")
		return
	}

	if err := h.db.UpdateSeries(id, req.Name, req.URL); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "更新剧集失败: "+err.Error())
		return
	}

	series, err := h.db.GetSeriesByID(id)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "获取更新后的剧集失败: "+err.Error())
		return
	}

	h.successResponse(w, series)
}

// DeleteSeries 删除剧集
func (h *Handler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.db.DeleteSeries(id); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "删除剧集失败: "+err.Error())
		return
	}

	h.successResponse(w, map[string]string{"message": "删除成功"})
}

// MarkAsWatched 标记为已观看
func (h *Handler) MarkAsWatched(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.db.MarkAsWatched(id); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "标记失败: "+err.Error())
		return
	}

	h.successResponse(w, map[string]string{"message": "标记为已观看"})
}

// MarkAsUnwatched 标记为未观看
func (h *Handler) MarkAsUnwatched(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.db.MarkAsUnwatched(id); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "标记失败: "+err.Error())
		return
	}

	h.successResponse(w, map[string]string{"message": "标记为未观看"})
}

// ToggleTracking 切换追踪状态
func (h *Handler) ToggleTracking(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.db.ToggleTracking(id); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "切换追踪状态失败: "+err.Error())
		return
	}

	series, err := h.db.GetSeriesByID(id)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "获取剧集信息失败: "+err.Error())
		return
	}

	status := "启用"
	if !series.IsTracking {
		status = "禁用"
	}

	h.successResponse(w, map[string]interface{}{
		"message":     fmt.Sprintf("追踪状态已%s", status),
		"is_tracking": series.IsTracking,
	})
}

// ClearSeriesHistory 清空剧集历史和当前进度
func (h *Handler) ClearSeriesHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.db.ClearSeriesHistory(id); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "清空历史失败: "+err.Error())
		return
	}

	series, err := h.db.GetSeriesByID(id)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "获取剧集信息失败: "+err.Error())
		return
	}

	h.successResponse(w, series)
}

// GetSettings 获取全局配置
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.db.GetSettings()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "获取配置失败: "+err.Error())
		return
	}
	h.successResponse(w, settings)
}

// UpdateSettings 更新全局配置
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings database.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 简单的验证
	if (settings.CrawlerStartTime != "" && !isValidTimeFormat(settings.CrawlerStartTime)) ||
		(settings.CrawlerEndTime != "" && !isValidTimeFormat(settings.CrawlerEndTime)) {
		h.errorResponse(w, http.StatusBadRequest, "时间格式不正确，请使用 HH:mm 格式")
		return
	}

	if err := h.db.UpdateSettings(&settings); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "更新配置失败: "+err.Error())
		return
	}
	h.successResponse(w, settings)
}

// TestSlackWebhook 测试 Slack Webhook
func (h *Handler) TestSlackWebhook(w http.ResponseWriter, r *http.Request) {
	// 创建测试消息
	testMessage := "🧪 这是一条来自 MiniCatch 的测试消息\n\n" +
		"时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n" +
		"如果您看到这条消息，说明 Slack Webhook 配置正确！"

	// 发送测试消息
	err := h.notifier.SendMessage(testMessage)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "发送测试消息失败: "+err.Error())
		return
	}

	h.successResponse(w, map[string]string{"message": "测试消息发送成功"})
}

// isValidTimeFormat 检查时间是否为 HH:mm 格式
func isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

// isCrawlerInWorkingHours 检查当前是否在爬虫工作时间段内
func (h *Handler) isCrawlerInWorkingHours() (bool, error) {
	settings, err := h.db.GetSettings()
	if err != nil {
		// 如果获取配置失败，默认允许执行，但返回错误以供记录
		return true, fmt.Errorf("获取配置失败: %v", err)
	}

	if settings.CrawlerStartTime == "" || settings.CrawlerEndTime == "" {
		// 如果没有设置时间，默认一直为工作时间
		return true, nil
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return true, fmt.Errorf("加载时区失败: %v", err)
	}
	nowInLoc := time.Now().In(loc)

	startTime, errStart := time.ParseInLocation("15:04", settings.CrawlerStartTime, loc)
	if errStart != nil {
		return true, fmt.Errorf("解析开始时间失败: %v", errStart)
	}

	endTime, errEnd := time.ParseInLocation("15:04", settings.CrawlerEndTime, loc)
	if errEnd != nil {
		return true, fmt.Errorf("解析结束时间失败: %v", errEnd)
	}

	// 将今天的日期应用到开始和结束时间上
	todayStart := time.Date(nowInLoc.Year(), nowInLoc.Month(), nowInLoc.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
	todayEnd := time.Date(nowInLoc.Year(), nowInLoc.Month(), nowInLoc.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)

	// 如果结束时间早于开始时间，说明是跨天的（例如 22:00 - 02:00）
	if todayEnd.Before(todayStart) {
		// 当前时间晚于今天的开始时间，或者早于今天的结束时间（意味着是第二天的凌晨）
		if nowInLoc.After(todayStart) || nowInLoc.Before(todayEnd) {
			return true, nil
		}
	} else {
		// 正常情况（例如 08:00 - 22:00）
		if nowInLoc.After(todayStart) && nowInLoc.Before(todayEnd) {
			return true, nil
		}
	}

	// 不在工作时间段内
	return false, nil
}

// HandleFetchTask 爬虫任务接口 - GET
func (h *Handler) HandleFetchTask(w http.ResponseWriter, r *http.Request) {
	inWorkingHours, err := h.isCrawlerInWorkingHours()
	if err != nil {
		// 检查工作时间出错，记录日志但默认放行
		log.Printf("检查爬虫工作时间出错: %v", err)
	}

	if !inWorkingHours {
		log.Printf("当前为爬虫非工作时间，不返回任务")
		h.successResponse(w, database.FetchTask{URLs: []string{}})
		return
	}

	// 获取所有启用的剧集URL
	urls, err := h.db.GetAllTrackingURLs()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "获取任务失败: "+err.Error())
		return
	}

	task := database.FetchTask{
		URLs: urls,
	}

	h.successResponse(w, task)
}

// HandleFetchTaskCallback 爬虫回调接口 - POST
func (h *Handler) HandleFetchTaskCallback(w http.ResponseWriter, r *http.Request) {
	var callback database.FetchCallback
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	log.Printf("收到爬虫回调: status=%d, message=%s, results=%d",
		callback.Status, callback.Message, len(callback.Results))

	if callback.Status >= 0 {
		// 处理成功的结果
		for _, result := range callback.Results {
			// 获取现有剧集信息
			series, err := h.db.GetSeriesByURL(result.URL)
			if err != nil {
				log.Printf("获取剧集信息失败 [%s]: %v", result.Name, err)
				continue
			}

			// 检查是否有新的集数
			existingSeries := make(map[string]bool)
			for _, ep := range series.History {
				existingSeries[ep] = true
			}

			var newEpisodes []string
			for _, ep := range result.Series {
				if !existingSeries[ep] {
					newEpisodes = append(newEpisodes, ep)
				}
			}

			if len(newEpisodes) > 0 { // 发现新集数
				log.Printf("📤 发现新集数: %s, %v", result.Name, newEpisodes)

				// 如果集数更新但是摘要没更新，那么不发送通知
				if result.Update != "" && result.Update == series.Current {
					log.Printf("摘要存在且没有更新，不发送通知")
				} else {
					go h.notifier.SendNotification(result.Name, newEpisodes, result.URL)
				}

				// 更新数据库
				if err := h.db.UpdateSeriesInfo(result.URL, result.Update, result.Series); err != nil {
					log.Printf("更新剧集信息失败 [%s]: %v", result.Name, err)
				}
			} else if result.Update != series.Current { // 发现新摘要
				log.Printf("📤 发现更新状态变更: %s, %s -> %s", result.Name, series.Current, result.Update)

				// 发送通知
				go h.notifier.SendStatusUpdateNotification(result.Name, series.Current, result.Update, result.URL)

				// 更新数据库
				if err := h.db.UpdateSeriesInfo(result.URL, result.Update, series.History); err != nil {
					log.Printf("更新剧集信息失败 [%s]: %v", result.Name, err)
				}
			} else { // 没有更新
				// 更新爬虫最后更新时间
				if err := h.db.UpdateSeriesCrawlerLastSeen(result.URL, time.Now()); err != nil {
					log.Printf("更新剧集爬虫最后更新时间失败 [%s]: %v", result.Name, err)
				}
			}
		}

		h.successResponse(w, map[string]string{"message": "OK"})
	} else {
		// 处理失败
		log.Printf("爬虫任务失败: %s", callback.Message)
		h.errorResponse(w, http.StatusBadRequest, "FAILED: "+callback.Message)
	}
}

// 错误响应
func (h *Handler) errorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: message,
	})
}

// 成功响应
func (h *Handler) successResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}
