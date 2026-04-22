package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xykcb_server/internal/cache"
	"xykcb_server/internal/config"
	"xykcb_server/internal/handler"
	"xykcb_server/internal/metrics"
	_ "xykcb_server/internal/provider/schools"
)

var srv *http.Server
var metricsStopCh = make(chan struct{})

func startServer(cfg *config.Config) {
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

	wrappedHandler := handler.Adapt(
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
		handler.RequestIDMiddleware(),
		handler.LoggingMiddleware(),
		handler.CORSMiddleware(),
		handler.RateLimiterMiddleware(rateLimit, rateWindow),
	)

	mux.Handle("/", wrappedHandler)

	srv = &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.HttpReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.HttpWriteTimeout) * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("服务器错误: %v", err)
		}
	}()
	log.Printf("服务器启动于端口 %s", cfg.Server.Port)
}

func stopServer() {
	if srv != nil {
		srv.Close()
	}
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := config.LoadNotFoundHTML(); err != nil {
		log.Fatalf("加载404页面失败: %v", err)
	}

	cache.StartTokenCacheCleanup(time.Minute)
	log.Println("Token缓存清理已启动")

	go metrics.StartReporter(time.Minute, metricsStopCh)
	log.Println("指标上报已启动")

	startServer(cfg)

	config.WatchAssets(
		func() {
			log.Println("正在更新配置...")
			newCfg, err := config.LoadConfig()
			if err != nil {
				log.Printf("更新配置失败: %v", err)
				return
			}
			stopServer()
			time.Sleep(time.Second)
			startServer(newCfg)
		},
		func() {
			log.Println("正在更新学校配置...")
			if err := config.LoadSchoolConfig(); err != nil {
				log.Printf("更新学校配置失败: %v", err)
			}
		},
		func() { log.Println("正在更新错误页面...") },
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("关闭服务器...")
	cache.StopTokenCacheCleanup()
	close(metricsStopCh)
	stopServer()
}
