package schools

import (
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

// HnitA is a school provider
type HnitA struct{}

func init() {
	provider.Default().Register(&HnitA{})
}

func (s *HnitA) GetSchoolId() string {
	return "1"
}

func (s *HnitA) GetNameZhcn() string {
	return "湖南工学院（移动端）"
}

func (s *HnitA) GetNameEn() string {
	return "Hunan Institute Of Technology (Mobile)"
}

func (s *HnitA) Login(account, password string) (*model.CourseResponse, error) {
	return &model.CourseResponse{
		Success: true,
		Data:    []interface{}{},
	}, nil
}
