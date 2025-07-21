package handlers

import (
	"net/http"

	"mini-catch/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(config *config.Config, handler *Handler) *chi.Mux {
	r := chi.NewRouter()

	// 中间件
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.GetHead)

	// 认证中间件
	r.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(config, next)
	})

	// API 路由
	r.Route("/api", func(r chi.Router) {
		// 登录接口（不需要认证）
		r.Post("/login", handler.LoginHandler)

		// 剧集管理接口
		r.Get("/series", handler.GetSeriesList)
		r.Post("/series", handler.CreateSeries)
		r.Put("/series/{id}", handler.UpdateSeries)
		r.Delete("/series/{id}", handler.DeleteSeries)
		r.Post("/series/{id}/watch", handler.MarkAsWatched)
		r.Post("/series/{id}/unwatch", handler.MarkAsUnwatched)
		r.Post("/series/{id}/toggle-tracking", handler.ToggleTracking)
		r.Post("/series/{id}/clear-history", handler.ClearSeriesHistory)

		// 爬虫接口
		r.Route("/fetch", func(r chi.Router) {
			r.Get("/", handler.HandleFetchTask)
			r.Post("/", handler.HandleFetchTaskCallback)
		})
	})

	// 静态文件服务
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/*", fileServer)

	return r
}
