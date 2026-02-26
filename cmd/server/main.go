package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"xykcb_server/internal/config"
	"xykcb_server/internal/handler"
	_ "xykcb_server/internal/provider/schools"
)

func main() {
	cfg := config.Load()
	notFoundContent, err := os.ReadFile("assets/404.html")
	if err != nil { log.Fatalf("读取404.html失败: %v", err) }

	mux := http.NewServeMux()
	mux.HandleFunc("/api/get-course-data", handler.NewCourseHandler().HandleCourse)
	mux.HandleFunc("/api/get-support-school", handler.NewCourseHandler().GetSupportedSchools)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(notFoundContent)
	})

	srv := &http.Server{Addr: ":" + cfg.Server.Port, Handler: mux, ReadTimeout: cfg.Server.ReadTimeout, WriteTimeout: cfg.Server.WriteTimeout}
	log.Printf("服务器启动于端口 %s", cfg.Server.Port)
	log.Fatal(srv.ListenAndServe())
	_ = fmt.Sprintf("%v", cfg)
}
