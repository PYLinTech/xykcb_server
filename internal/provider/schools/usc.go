package schools

import (
	"xykcb_server/internal/config"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type Usc struct{}

func init() { provider.Default().Register(&Usc{}) }

func (s *Usc) GetSchoolId() string    { return "4" }
func (s *Usc) GetProviderKey() string { return "usc" }

func (s *Usc) GetSchoolConfig() *config.SchoolSemesters {
	return config.GetSchoolConfigById(s.GetProviderKey())
}

func (s *Usc) Login(account, password string) (*model.CourseResponse, error) {
	// TODO: 实现登录逻辑
	return &model.CourseResponse{Success: true, Data: nil}, nil
}
