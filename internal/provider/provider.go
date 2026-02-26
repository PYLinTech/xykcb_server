package provider

import "xykcb_server/internal/model"

type SchoolProvider interface {
	GetSchoolId() string
	GetNameZhcn() string
	GetNameEn() string
	Login(account, password string) (*model.CourseResponse, error)
}

type Registry struct{ providers map[string]SchoolProvider }

var defaultRegistry = &Registry{providers: make(map[string]SchoolProvider)}

func Default() *Registry { return defaultRegistry }
func NewRegistry() *Registry { return &Registry{providers: make(map[string]SchoolProvider)} }
func (r *Registry) Register(p SchoolProvider) { r.providers[p.GetSchoolId()] = p }
func (r *Registry) Get(school string) (SchoolProvider, bool) { p, ok := r.providers[school]; return p, ok }

type SchoolInfo struct {
	Id       string `json:"id"`
	NameZhcn string `json:"name_zhcn"`
	NameEn   string `json:"name_en"`
}

func (r *Registry) ListAll() []SchoolInfo {
	infos := make([]SchoolInfo, 0, len(r.providers))
	for _, p := range r.providers { infos = append(infos, SchoolInfo{Id: p.GetSchoolId(), NameZhcn: p.GetNameZhcn(), NameEn: p.GetNameEn()}) }
	return infos
}
