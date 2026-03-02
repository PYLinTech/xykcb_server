package schools

import (
	"xykcb_server/internal/config"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type HnitB struct{}

func init() { provider.Default().Register(&HnitB{}) }

func (s *HnitB) GetSchoolId() string    { return "2" }
func (s *HnitB) GetProviderKey() string { return "hnit_b" }

func (s *HnitB) GetSchoolConfig() *config.SchoolSemesters {
	return config.GetSchoolConfigById(s.GetProviderKey())
}

func (s *HnitB) Login(account, password string) (*model.CourseResponse, error) {
	// TODO: 实现登录逻辑
	return &model.CourseResponse{Success: true, Data: nil}, nil
}
