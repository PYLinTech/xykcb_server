package schools

import (
	"sync"

	"xykcb_server/internal/config"
	"xykcb_server/internal/model"
)

func (s *HnitA) Login(account, password string) (*model.CourseResponse, error) {
	token, err := s.getToken(account, password)
	if err != nil {
		return s.error(err.Error()), nil
	}

	schoolCfg := s.GetSchoolConfig()
	if schoolCfg == nil {
		return s.error(""), nil
	}
	semesterConfigs := s.getSemesterConfigs(account, password, token, schoolCfg)
	if len(semesterConfigs) == 0 {
		return s.error(""), nil
	}

	result := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for semesterID, semesterConfig := range semesterConfigs {
		wg.Add(1)
		go func(semesterID string, semesterConfig config.SemesterConfig) {
			defer wg.Done()

			curriculumData, err := s.retryWithValidToken(account, password, "/student/curriculum?token="+token+"&xnxq01id="+semesterID+"&week=all", func(path string) (map[string]interface{}, error) {
				return schoolClient.Get(path)
			})
			if err != nil || curriculumData == nil {
				return
			}

			var courses []map[string]interface{}
			if semesterData, ok := curriculumData["data"].([]interface{}); ok {
				for _, item := range semesterData {
					if courseList, ok := item.(map[string]interface{})["item"].([]interface{}); ok {
						for _, c := range courseList {
							if course, ok := c.(map[string]interface{}); ok {
								courses = append(courses, convertCourse(course))
							}
						}
					}
				}
			}
			courses = resolveDuplicateCourseIDs(courses)

			mu.Lock()
			result[semesterID] = map[string]interface{}{
				"semesterStart":     semesterConfig.SemesterStart,
				"totalWeeks":        semesterConfig.TotalWeeks,
				"timeSlots":         semesterConfig.TimeSlots,
				"mergeableSections": semesterConfig.MergeableSections,
				"courses":           courses,
			}
			mu.Unlock()
		}(semesterID, semesterConfig)
	}

	wg.Wait()

	return &model.CourseResponse{Success: true, Data: result}, nil
}
