package config

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port             string `json:"port"`
	HttpReadTimeout  int    `json:"httpReadTimeout"`
	HttpWriteTimeout int    `json:"httpWriteTimeout"`
}

type CORSConfig struct {
	AllowAll     bool     `json:"allowAll"`
	AllowedHosts []string `json:"allowedHosts"`
}

type Config struct {
	Server ServerConfig `json:"server"`
	CORS   CORSConfig   `json:"cors"`
}

type TimeSlot struct {
	Section int    `json:"section"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type SemesterConfig struct {
	SemesterStart string     `json:"semesterStart"`
	TotalWeeks    int        `json:"totalWeeks"`
	TimeSlots     []TimeSlot `json:"timeSlots"`
}

type SchoolSemesters struct {
	Semesters  map[string]SemesterConfig `json:"semesters"`
	Functions  interface{}               `json:"functions"`
}

type SchoolConfig map[string]SchoolSemesters

// 文件缓存结构
type fileCache struct {
	modTime time.Time
	content []byte
}

var (
	serverV      *viper.Viper
	schoolV     *viper.Viper
	serverCfg    *Config
	schoolCfg    SchoolConfig
	NotFoundHTML []byte
	watcher      *fsnotify.Watcher
	mu           sync.Mutex
	// 文件缓存
	configCache, schoolCache, notFoundCache fileCache
)

func LoadConfig() (*Config, error) {
	serverV = viper.New()
	serverV.SetConfigName("config")
	serverV.AddConfigPath("assets")
	serverV.SetConfigType("json")

	if err := serverV.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := serverV.Unmarshal(&serverCfg); err != nil {
		return nil, err
	}

	if serverCfg.Server.Port == "" {
		serverCfg.Server.Port = "8080"
	}
	if serverCfg.Server.HttpReadTimeout <= 0 {
		serverCfg.Server.HttpReadTimeout = 30
	}
	if serverCfg.Server.HttpWriteTimeout <= 0 {
		serverCfg.Server.HttpWriteTimeout = 30
	}

	updateCache("assets/config.json", &configCache)
	return serverCfg, nil
}

func LoadNotFoundHTML() error {
	var err error
	NotFoundHTML, err = os.ReadFile("assets/404.html")
	if err != nil {
		return err
	}
	updateCache("assets/404.html", &notFoundCache)
	return nil
}

func LoadSchoolConfig() error {
	schoolV = viper.New()
	schoolV.SetConfigName("school_config")
	schoolV.AddConfigPath("assets")
	schoolV.SetConfigType("json")

	if err := schoolV.ReadInConfig(); err != nil {
		return err
	}
	schoolV.Unmarshal(&schoolCfg)
	updateCache("assets/school_config.json", &schoolCache)
	return nil
}

// 更新文件缓存
func updateCache(path string, cache *fileCache) {
	if data, err := os.ReadFile(path); err == nil {
		if stat, err := os.Stat(path); err == nil {
			cache.modTime = stat.ModTime()
			cache.content = data
		}
	}
}

func GetSchoolConfigById(id string) *SchoolSemesters {
	if s, ok := schoolCfg[id]; ok {
		return &s
	}
	return nil
}

func GetSchoolFunctionsById(id string) interface{} {
	if s, ok := schoolCfg[id]; ok {
		return s.Functions
	}
	return nil
}

func GetCORSConfig() *CORSConfig {
	if serverCfg == nil {
		return &CORSConfig{AllowAll: true}
	}
	return &serverCfg.CORS
}

func WatchAssets(onConfigChange, onSchoolChange, onNotFoundChange func()) {
	watcher, _ = fsnotify.NewWatcher()
	go func() {
		defer watcher.Close()
		watcher.Add("assets")
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					switch event.Name {
					case "assets/404.html":
						if checkChange(event.Name, &notFoundCache) {
							loadAndNotify(LoadNotFoundHTML, onNotFoundChange)
						}
					case "assets/config.json":
						if checkChange(event.Name, &configCache) {
							onConfigChange()
						}
					case "assets/school_config.json":
						if checkChange(event.Name, &schoolCache) {
							onSchoolChange()
						}
					}
				}
			case err := <-watcher.Errors:
				_ = err
			}
		}
	}()
}

// 加载并通知，失败时记录日志
func loadAndNotify(loadFunc func() error, notify func()) {
	if err := loadFunc(); err != nil {
		log.Printf("更新失败: %v", err)
		return
	}
	notify()
}

// 检查文件是否有变化：修改时间变化 且 内容变化
func checkChange(path string, cache *fileCache) bool {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}

	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	currentModTime := stat.ModTime()

	// 首次加载：记录状态，不触发更新
	if cache.modTime.IsZero() {
		cache.modTime = currentModTime
		cache.content = data
		return false
	}

	// 两者都变化才触发
	if currentModTime.After(cache.modTime) && string(data) != string(cache.content) {
		cache.modTime = currentModTime
		cache.content = data
		return true
	}
	return false
}
