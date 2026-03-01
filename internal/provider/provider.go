package provider

import (
	"log"

	"xykcb_server/internal/config"
	"xykcb_server/internal/model"
)

type SchoolProvider interface {
	GetSchoolId() string
	GetProviderKey() string
	Login(account, password string) (*model.CourseResponse, error)
}

type Registry struct {
	providers map[string]SchoolProvider
}

var defaultRegistry = &Registry{providers: make(map[string]SchoolProvider)}

func init() {
	if err := config.LoadSchoolConfig(); err != nil {
		log.Printf("加载学校配置失败: %v", err)
	}
}

func Default() *Registry { return defaultRegistry }

func (r *Registry) Register(p SchoolProvider) { r.providers[p.GetProviderKey()] = p }

func (r *Registry) Get(school string) (SchoolProvider, bool) {
	p, ok := r.providers[school]
	return p, ok
}

type SchoolInfo struct {
	Id      string `json:"id"`
	DescKey string `json:"desc_key"`
}

func (r *Registry) ListAll() []SchoolInfo {
	infos := make([]SchoolInfo, 0, len(r.providers))
	for _, p := range r.providers {
		infos = append(infos, SchoolInfo{
			Id:      p.GetSchoolId(),
			DescKey: p.GetProviderKey(),
		})
	}
	return infos
}
