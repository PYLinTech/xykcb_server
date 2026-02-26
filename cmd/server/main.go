package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xykcb_server/internal/config"
	"xykcb_server/internal/handler"
	_ "xykcb_server/internal/provider/schools"
)

var srv *http.Server

func startServer(cfg *config.Config) {
	mux := http.NewServeMux()
	courseHandler := handler.NewCourseHandler()
	mux.HandleFunc("/api/get-course-data", courseHandler.HandleCourse)
	mux.HandleFunc("/api/get-support-school", courseHandler.GetSupportedSchools)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(config.NotFoundHTML)
	})

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
	stopServer()
}
