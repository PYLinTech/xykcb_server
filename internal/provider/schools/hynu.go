package schools

import (
	"xykcb_server/internal/config"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type Hynu struct{}

func init() { provider.Default().Register(&Hynu{}) }

func (s *Hynu) GetSchoolId() string    { return "3" }
func (s *Hynu) GetProviderKey() string { return "hynu" }

func (s *Hynu) GetSchoolConfig() *config.SchoolSemesters {
	return config.GetSchoolConfigById(s.GetProviderKey())
}

func (s *Hynu) Login(account, password string) (*model.CourseResponse, error) {
	// TODO: 实现登录逻辑
	return &model.CourseResponse{Success: true, Data: nil}, nil
}
