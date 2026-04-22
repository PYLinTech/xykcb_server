package schools

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"xykcb_server/internal/cache"
	"xykcb_server/internal/config"
	"xykcb_server/internal/httpclient"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

var schoolClient = httpclient.NewClient("https://jw.hnit.edu.cn/njwhd", 10*time.Second)

type HnitA struct{}

var locationRe = regexp.MustCompile(`\(.*?\)|（.*?）`)

func init() { provider.Default().Register(&HnitA{}) }

func (s *HnitA) GetSchoolId() string    { return "1" }
func (s *HnitA) GetProviderKey() string { return "hnit_a" }

func (s *HnitA) GetSchoolConfig() *config.SchoolSemesters {
	return config.GetSchoolConfigById(s.GetProviderKey())
}

func (s *HnitA) error(msg string) *model.CourseResponse {
	if strings.Contains(msg, "密码错误") {
		return &model.CourseResponse{Success: false, DescKey: "003"}
	}
	return &model.CourseResponse{Success: false, DescKey: "004"}
}

func cleanLocation(location string) string {
	return locationRe.ReplaceAllString(location, "")
}

func parseClassTime(ct string) map[string][]int {
	if len(ct) < 3 {
		return nil
	}
	schedule := make(map[string][]int, 1)
	weekday, sections := string(ct[0]), ct[1:]
	schedule[weekday] = make([]int, 0, len(sections)/2)
	for i := 0; i+1 < len(sections); i += 2 {
		section, _ := strconv.Atoi(sections[i : i+2])
		schedule[weekday] = append(schedule[weekday], section)
	}
	return schedule
}

func parseWeeks(wd string) []int {
	fields := strings.Split(wd, ",")
	weeks := make([]int, 0, len(fields))
	for _, p := range fields {
		if p == "" {
			continue
		}
		if w, err := strconv.Atoi(p); err == nil {
			weeks = append(weeks, w)
		}
	}
	return weeks
}

func safeString(v interface{}, def string) string {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

func safeStringMap(v interface{}, key string) string {
	if v == nil {
		return ""
	}
	if m, ok := v.(map[string]interface{}); ok {
		return safeString(m[key], "")
	}
	return ""
}

func convertCourse(c map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":       safeString(c["kch"], ""),
		"name":     safeString(c["courseName"], ""),
		"location": cleanLocation(safeString(c["location"], "")),
		"teacher":  safeString(c["teacherName"], ""),
		"weeks":    parseWeeks(safeString(c["classWeekDetails"], "")),
		"schedule": parseClassTime(safeString(c["classTime"], "")),
	}
}

func (s *HnitA) retryWithValidToken(account, password, path string, fetch func(path string) (map[string]interface{}, error)) (map[string]interface{}, error) {
	data, err := fetch(path)
	if err != nil {
		return nil, err
	}

	code := safeString(data["code"], "")
	if code != "" && code != "1" {
		cache.InvalidateToken(s.GetProviderKey(), account)
		token, err := s.getToken(account, password)
		if err != nil {
			return nil, err
		}
		replacedPath := httpclient.ReplaceTokenInPath(path, token)
		return fetch(replacedPath)
	}

	return data, nil
}

func (s *HnitA) encryptPassword(password string) string {
	key, _ := hex.DecodeString("717a6b6a316b6a6768643d383736262a")
	block, _ := aes.NewCipher(key)
	plain := []byte("\"" + password + "\"")
	size := block.BlockSize()
	padding := size - len(plain)%size
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	encrypted := make([]byte, len(plain))
	for i := 0; i < len(plain); i += size {
		block.Encrypt(encrypted[i:i+size], plain[i:i+size])
	}
	return base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(encrypted)))
}

func (s *HnitA) getToken(account, password string) (string, error) {
	return cache.GetToken(s.GetProviderKey(), account, password, func(account, password string) (string, error) {
		resp, err := schoolClient.Post("/login?userNo="+account+"&pwd="+s.encryptPassword(password), "")
		if err != nil {
			return "", err
		}

		code := safeString(resp["code"], "")
		if code != "1" {
			return "", fmt.Errorf("%s", safeString(resp["Msg"], "login failed"))
		}

		token := safeStringMap(resp["data"], "token")
		if token == "" {
			return "", fmt.Errorf("no token in response")
		}

		return token, nil
	})
}

func (s *HnitA) Login(account, password string) (*model.CourseResponse, error) {
	token, err := s.getToken(account, password)
	if err != nil {
		return s.error(err.Error()), nil
	}

	schoolCfg := s.GetSchoolConfig()
	if schoolCfg == nil {
		return s.error(""), nil
	}

	result := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for semesterID, semesterConfig := range schoolCfg.Semesters {
		wg.Add(1)
		go func(semesterID string, semesterConfig config.SemesterConfig) {
			defer wg.Done()

			curriculumData, err := s.retryWithValidToken(account, password, "/student/curriculum?token="+token+"&xnxq01id="+semesterID+"&week=all", func(path string) (map[string]interface{}, error) {
				return schoolClient.Get(path, nil)
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

func (s *HnitA) GetGrades(account, password, semester string) (*model.CourseResponse, error) {
	token, err := s.getToken(account, password)
	if err != nil {
		return s.error(err.Error()), nil
	}

	semesterListData, err := s.retryWithValidToken(account, password, "/semesterList?token="+token, func(path string) (map[string]interface{}, error) {
		return schoolClient.Get(path, nil)
	})
	if err != nil || semesterListData == nil {
		return s.error(err.Error()), nil
	}

	code := safeString(semesterListData["code"], "")
	if code != "1" {
		return s.error(safeString(semesterListData["Msg"], "")), nil
	}

	gradesData, err := s.retryWithValidToken(account, password, "/student/termGPA?token="+token+"&semester="+semester, func(path string) (map[string]interface{}, error) {
		return schoolClient.Get(path, nil)
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
		return schoolClient.Get(path, nil)
	})
	if err != nil || guidanceData == nil {
		return s.error(err.Error()), nil
	}

	if safeString(guidanceData["code"], "") != "1" {
		return s.error(safeString(guidanceData["Msg"], "")), nil
	}

	return &model.CourseResponse{Success: true, Data: guidanceData["data"]}, nil
}

func fetchURL(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data, err
}
