package schools

import (
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type stubSchool struct {
	id  string
	key string
}

func init() {
	provider.Default().Register(&stubSchool{id: "2", key: "hnit_b"})
	provider.Default().Register(&stubSchool{id: "3", key: "hynu"})
	provider.Default().Register(&stubSchool{id: "4", key: "usc"})
}

func (s *stubSchool) GetSchoolId() string    { return s.id }
func (s *stubSchool) GetProviderKey() string { return s.key }

func (s *stubSchool) Login(account, password string) (*model.CourseResponse, error) {
	return &model.CourseResponse{Success: true, Data: nil}, nil
}

func (s *stubSchool) GetGrades(account, password, semester string) (*model.CourseResponse, error) {
	return &model.CourseResponse{Success: false, DescKey: "004"}, nil
}

func (s *stubSchool) GetGuidanceTeaching(account, password string) (*model.CourseResponse, error) {
	return &model.CourseResponse{Success: false, DescKey: "004"}, nil
}
