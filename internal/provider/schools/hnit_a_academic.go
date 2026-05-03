package schools

import "xykcb_server/internal/model"

func (s *HnitA) GetGrades(account, password, semester string) (*model.CourseResponse, error) {
	token, err := s.getToken(account, password)
	if err != nil {
		return s.error(err.Error()), nil
	}

	semesterListData, err := s.retryWithValidToken(account, password, "/semesterList?token="+token, func(path string) (map[string]interface{}, error) {
		return schoolClient.Get(path)
	})
	if err != nil || semesterListData == nil {
		return s.error(err.Error()), nil
	}

	code := safeString(semesterListData["code"], "")
	if code != "1" {
		return s.error(safeString(semesterListData["Msg"], "")), nil
	}

	gradesData, err := s.retryWithValidToken(account, password, "/student/termGPA?token="+token+"&semester="+semester, func(path string) (map[string]interface{}, error) {
		return schoolClient.Get(path)
	})
	if err != nil || gradesData == nil {
		return s.error(err.Error()), nil
	}

	if safeString(gradesData["code"], "") != "1" {
		return s.error(safeString(gradesData["Msg"], "")), nil
	}

	newData := map[string]interface{}{
		"all-semester": semesterListData["data"],
		"all-grades":   gradesData["data"],
	}

	return &model.CourseResponse{Success: true, Data: newData}, nil
}

func (s *HnitA) GetGuidanceTeaching(account, password string) (*model.CourseResponse, error) {
	token, err := s.getToken(account, password)
	if err != nil {
		return s.error(err.Error()), nil
	}

	guidanceData, err := s.retryWithValidToken(account, password, "/student/guidanceTeaching?token="+token, func(path string) (map[string]interface{}, error) {
		return schoolClient.Get(path)
	})
	if err != nil || guidanceData == nil {
		return s.error(err.Error()), nil
	}

	if safeString(guidanceData["code"], "") != "1" {
		return s.error(safeString(guidanceData["Msg"], "")), nil
	}

	return &model.CourseResponse{Success: true, Data: guidanceData["data"]}, nil
}
