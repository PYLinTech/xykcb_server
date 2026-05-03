package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xykcb_server/internal/app"
	"xykcb_server/internal/cache"
	"xykcb_server/internal/config"
	_ "xykcb_server/internal/provider/schools"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := config.LoadNotFoundHTML(); err != nil {
		log.Fatalf("加载404页面失败: %v", err)
	}
	if err := config.LoadSchoolConfig(); err != nil {
		log.Fatalf("加载学校配置失败: %v", err)
	}

	cache.StartTokenCacheCleanup(time.Minute)
	log.Println("Token缓存清理已启动")

	server := app.New()
	server.Start(cfg)
	server.WatchAssets()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("关闭服务器...")
	cache.StopTokenCacheCleanup()
	server.Stop()
}
