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

	"github.com/go-chi/chi/v5"
)

// Handler HTTP处理器
type Handler struct {
	db       *database.Database
	config   config.Config
	notifier *slack.Notifier
}

// NewHandler 创建新的处理器
func NewHandler(db *database.Database, authConfig config.Config, notifier *slack.Notifier) *Handler {
	return &Handler{
		db:       db,
		config:   authConfig,
		notifier: notifier,
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

	// 验证用户名和密码
	if req.Username != h.config.Auth.Username || req.Password != h.config.Auth.Password {
		h.errorResponse(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 生成认证令牌
	token := GenerateAuthToken(req.Username, req.Password)

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

// HandleFetchTask 爬虫任务接口 - GET
func (h *Handler) HandleFetchTask(w http.ResponseWriter, r *http.Request) {
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

			if len(newEpisodes) > 0 {
				log.Printf("📤 发现新集数: %s, %v", result.Name, newEpisodes)
				// 发送Slack通知
				go h.notifier.SendNotification(result.Name, newEpisodes, result.URL)

				// 更新数据库
				if err := h.db.UpdateSeriesInfo(result.URL, result.Update, result.Series); err != nil {
					log.Printf("更新剧集信息失败 [%s]: %v", result.Name, err)
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
