package config

import (
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port            string `json:"port"`
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
	Semesters map[string]SemesterConfig `json:"semesters"`
}

type SchoolConfig map[string]SchoolSemesters

var (
	serverV      *viper.Viper
	schoolV     *viper.Viper
	serverCfg    *Config
	schoolCfg   SchoolConfig
	NotFoundHTML []byte
	watcher     *fsnotify.Watcher
	mu          sync.Mutex
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

	if serverCfg.Server.Port == "" { serverCfg.Server.Port = "8080" }
	if serverCfg.Server.HttpReadTimeout <= 0 { serverCfg.Server.HttpReadTimeout = 30 }
	if serverCfg.Server.HttpWriteTimeout <= 0 { serverCfg.Server.HttpWriteTimeout = 30 }

	log.Println("成功加载 config.json")
	return serverCfg, nil
}

func LoadNotFoundHTML() error {
	var err error
	NotFoundHTML, err = os.ReadFile("assets/404.html")
	if err != nil {
		return err
	}
	log.Println("成功加载 404.html")
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
	log.Println("成功加载 school_config.json")
	return nil
}

func GetSchoolConfigById(id string) *SchoolSemesters {
	if s, ok := schoolCfg[id]; ok {
		return &s
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
					if event.Name == "assets/404.html" {
						if err := LoadNotFoundHTML(); err != nil {
							log.Printf("更新错误页面失败: %v", err)
						} else {
							onNotFoundChange()
						}
					} else if isValidConfig(event.Name) {
						if event.Name == "assets/config.json" {
							onConfigChange()
						} else if event.Name == "assets/school_config.json" {
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

func isValidConfig(path string) bool {
	mu.Lock()
	defer mu.Unlock()

	var v *viper.Viper
	switch path {
	case "assets/config.json":
		v = serverV
	case "assets/school_config.json":
		v = schoolV
	}
	// 已加载的 viper 实例说明配置文件有效
	return v != nil
}
