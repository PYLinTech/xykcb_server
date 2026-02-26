package schools

import (
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

// HnitB is a school provider
type HnitB struct{}

func init() {
	provider.Default().Register(&HnitB{})
}

func (s *HnitB) GetSchoolId() string {
	return "2"
}

func (s *HnitB) GetNameZhcn() string {
	return "湖南工学院（PC端）"
}

func (s *HnitB) GetNameEn() string {
	return "Hunan Institute Of Technology (PC)"
}

func (s *HnitB) Login(account, password string) (*model.CourseResponse, error) {
	return &model.CourseResponse{
		Success: true,
		Data:    []interface{}{},
	}, nil
}
