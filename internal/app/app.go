package app

import (
	"log"
	"net/http"
	"time"

	"xykcb_server/internal/config"
	"xykcb_server/internal/handler"
)

type App struct {
	server *http.Server
}

func New() *App {
	return &App{}
}

func (a *App) Start(cfg *config.Config) {
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      a.routes(cfg),
		ReadTimeout:  time.Duration(cfg.Server.HttpReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.HttpWriteTimeout) * time.Second,
	}
	a.server = srv

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("服务器错误: %v", err)
		}
	}()
	log.Printf("服务器启动于端口 %s", cfg.Server.Port)
}

func (a *App) Stop() {
	if a.server != nil {
		a.server.Close()
	}
}

func (a *App) WatchAssets() {
	config.WatchAssets(
		func() {
			log.Println("正在更新配置...")
			newCfg, err := config.LoadConfig()
			if err != nil {
				log.Printf("更新配置失败: %v", err)
				return
			}
			a.Stop()
			time.Sleep(time.Second)
			a.Start(newCfg)
		},
		func() {
			log.Println("正在更新学校配置...")
			if err := config.LoadSchoolConfig(); err != nil {
				log.Printf("更新学校配置失败: %v", err)
			}
		},
		func() { log.Println("正在更新错误页面...") },
	)
}

func (a *App) routes(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	courseHandler := handler.NewCourseHandler()

	rateLimit := cfg.Server.RateLimit
	rateWindow := time.Duration(cfg.Server.RateWindow) * time.Second
	if rateLimit <= 0 {
		rateLimit = 1000
	}
	if rateWindow <= 0 {
		rateWindow = time.Minute
	}

	mux.Handle("/", handler.Adapt(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/get-course-data":
				courseHandler.HandleCourse(w, r)
			case "/get-course-grades":
				courseHandler.HandleCourseGrades(w, r)
			case "/get-guidance-teaching":
				courseHandler.HandleGuidanceTeaching(w, r)
			case "/get-support-school":
				courseHandler.GetSupportedSchools(w, r)
			case "/get-support-function":
				courseHandler.GetSupportFunctions(w, r)
			default:
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(config.NotFoundHTML)
			}
		}),
		handler.CORSMiddleware(),
		handler.RateLimiterMiddleware(rateLimit, rateWindow),
	))

	return mux
}
