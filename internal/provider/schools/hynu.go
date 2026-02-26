package schools

import (
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

// Hynu is a school provider
type Hynu struct{}

func init() {
	provider.Default().Register(&Hynu{})
}

func (s *Hynu) GetSchoolId() string {
	return "3"
}

func (s *Hynu) GetNameZhcn() string {
	return "衡阳师范学院"
}

func (s *Hynu) GetNameEn() string {
	return "Hengyang Normal University"
}

func (s *Hynu) Login(account, password string) (*model.CourseResponse, error) {
	return &model.CourseResponse{
		Success: true,
		Data:    []interface{}{},
	}, nil
}
