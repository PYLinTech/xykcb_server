package provider

import "xykcb_server/internal/model"

// SchoolProvider defines the interface for school-specific login implementation
type SchoolProvider interface {
	GetSchoolId() string
	GetNameZhcn() string
	GetNameEn() string
	Login(account, password string) (*model.CourseResponse, error)
}

// Registry holds all registered school providers
type Registry struct {
	providers map[string]SchoolProvider
}

var defaultRegistry = &Registry{
	providers: make(map[string]SchoolProvider),
}

// Default returns the default global registry
func Default() *Registry {
	return defaultRegistry
}

// NewRegistry creates a new provider registry (for testing)
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]SchoolProvider),
	}
}

// Register adds a new school provider to the registry
func (r *Registry) Register(provider SchoolProvider) {
	r.providers[provider.GetSchoolId()] = provider
}

// Get returns the provider for a given school id
func (r *Registry) Get(school string) (SchoolProvider, bool) {
	p, ok := r.providers[school]
	return p, ok
}

// SchoolInfo contains school information
type SchoolInfo struct {
	Id       string `json:"id"`
	NameZhcn string `json:"name_zhcn"`
	NameEn   string `json:"name_en"`
}

// ListAll returns all registered schools with their info
func (r *Registry) ListAll() []SchoolInfo {
	infos := make([]SchoolInfo, 0, len(r.providers))
	for _, p := range r.providers {
		infos = append(infos, SchoolInfo{
			Id:       p.GetSchoolId(),
			NameZhcn: p.GetNameZhcn(),
			NameEn:   p.GetNameEn(),
		})
	}
	return infos
}
