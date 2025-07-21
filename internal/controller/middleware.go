package handlers

import (
	"encoding/base64"
	"mini-catch/internal/config"
	"net/http"
	"strings"
)

// 认证中间件
func AuthMiddleware(config *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 跳过不需要认证的路径
		if shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 检查 Authorization 头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// 尝试从 Cookie 获取认证信息
			cookie, err := r.Cookie("auth_token")
			if err != nil || cookie.Value == "" {
				http.Error(w, "需要认证", http.StatusUnauthorized)
				return
			}
			authHeader = cookie.Value
		}

		// 验证认证信息
		if !validateAuth(authHeader, config) {
			http.Error(w, "认证失败", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// 验证认证信息
func validateAuth(authHeader string, config *config.Config) bool {
	// 移除 "Bearer " 前缀（如果存在）
	authHeader = strings.TrimPrefix(authHeader, "Bearer ")

	// 解码 base64
	decoded, err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		return false
	}

	// 解析用户名和密码
	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return false
	}

	username := parts[0]
	password := parts[1]

	// 验证用户名和密码
	return username == config.Auth.Username && password == config.Auth.Password
}

// shouldSkipAuth 判断是否需要跳过认证
func shouldSkipAuth(path string) bool {
	// 不需要认证的路径
	skipPaths := []string{
		"/api/login",   // 登录接口
		"/favicon.ico", // 网站图标
	}

	// 检查精确匹配
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	// 静态文件路径（以 / 开头但不是 API 路径）
	if path == "/" || (!strings.HasPrefix(path, "/api/")) {
		return true
	}

	return false
}

// 生成认证令牌
func GenerateAuthToken(username, password string) string {
	token := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(token))
}
