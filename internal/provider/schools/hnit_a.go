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
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true, // 禁用长连接，每个请求完成后立即断开
	},
}

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

func convertCourse(c map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":       c["kch"],
		"name":     c["courseName"],
		"location": cleanLocation(c["location"].(string)),
		"teacher":  c["teacherName"],
		"weeks":    parseWeeks(c["classWeekDetails"].(string)),
		"schedule": parseClassTime(c["classTime"].(string)),
	}
}

// retryWithValidToken 检查 token 失效并重试
func (s *HnitA) retryWithValidToken(account, password, url string, fetch func(url string) (map[string]interface{}, error)) (map[string]interface{}, error) {
	data, err := fetch(url)
	if err != nil {
		return nil, err
	}

	// 检查 token 是否失效
	if data["code"] != nil && data["code"].(string) != "1" {
		cache.InvalidateToken(s.GetProviderKey(), account)
		token, err := s.getToken(account, password)
		if err != nil {
			return nil, err
		}
		// 替换 url 中的 token
		replacedURL := strings.Replace(url, "token="+extractToken(url), "token="+token, 1)
		return fetch(replacedURL)
	}

	return data, nil
}

func extractToken(url string) string {
	if idx := strings.Index(url, "token="); idx != -1 {
		tokenPart := url[idx+7:]
		if endIdx := strings.Index(tokenPart, "&"); endIdx != -1 {
			return tokenPart[:endIdx]
		}
		return tokenPart
	}
	return ""
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

// getToken 获取 token，优先使用缓存，失败则重新登录
func (s *HnitA) getToken(account, password string) (string, error) {
	return cache.GetToken(s.GetProviderKey(), account, password, func(account, password string) (string, error) {
		resp, err := httpClient.Post("https://jw.hnit.edu.cn/njwhd/login?userNo="+account+"&pwd="+s.encryptPassword(password), "", nil)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return "", err
		}

		if data["code"].(string) != "1" {
			return "", fmt.Errorf(data["Msg"].(string))
		}

		return data["data"].(map[string]interface{})["token"].(string), nil
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

			curriculumResp, err := httpClient.Get("https://jw.hnit.edu.cn/njwhd/student/curriculum?token=" + token + "&xnxq01id=" + semesterID + "&week=all")
			if err != nil {
				return
			}
			defer curriculumResp.Body.Close()

			var curriculumData map[string]interface{}
			if err := json.NewDecoder(curriculumResp.Body).Decode(&curriculumData); err != nil {
				return
			}

			curriculumData, _ = s.retryWithValidToken(account, password, "https://jw.hnit.edu.cn/njwhd/student/curriculum?token="+token+"&xnxq01id="+semesterID+"&week=all", func(url string) (map[string]interface{}, error) {
				resp, err := httpClient.Get(url)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				var data map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&data)
				return data, err
			})
			if curriculumData == nil {
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

	// 请求 semesterList 接口
	semesterListResp, err := httpClient.Get("https://jw.hnit.edu.cn/njwhd/semesterList?token=" + token)
	if err != nil {
		return s.error(err.Error()), nil
	}
	defer semesterListResp.Body.Close()

	var semesterListData map[string]interface{}
	if err := json.NewDecoder(semesterListResp.Body).Decode(&semesterListData); err != nil {
		return s.error(err.Error()), nil
	}

	semesterListData, err = s.retryWithValidToken(account, password, "https://jw.hnit.edu.cn/njwhd/semesterList?token="+token, func(url string) (map[string]interface{}, error) {
		resp, err := httpClient.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var data map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&data)
		return data, err
	})
	if err != nil {
		return s.error(err.Error()), nil
	}

	if semesterListData["code"].(string) != "1" {
		return s.error(semesterListData["Msg"].(string)), nil
	}

	// 请求成绩接口
	url := "https://jw.hnit.edu.cn/njwhd/student/termGPA?token=" + token + "&semester=" + semester
	gradesResp, err := httpClient.Get(url)
	if err != nil {
		return s.error(err.Error()), nil
	}
	defer gradesResp.Body.Close()

	var gradesData map[string]interface{}
	if err := json.NewDecoder(gradesResp.Body).Decode(&gradesData); err != nil {
		return s.error(err.Error()), nil
	}

	gradesData, err = s.retryWithValidToken(account, password, "https://jw.hnit.edu.cn/njwhd/student/termGPA?token="+token+"&semester="+semester, func(url string) (map[string]interface{}, error) {
		resp, err := httpClient.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var data map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&data)
		return data, err
	})
	if err != nil {
		return s.error(err.Error()), nil
	}

	if gradesData["code"].(string) != "1" {
		return s.error(gradesData["Msg"].(string)), nil
	}

	// 拼合新的 data: all-semester 和 all-grades
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

	// 请求 guidanceTeaching 接口
	guidanceResp, err := httpClient.Get("https://jw.hnit.edu.cn/njwhd/student/guidanceTeaching?token=" + token)
	if err != nil {
		return s.error(err.Error()), nil
	}
	defer guidanceResp.Body.Close()

	var guidanceData map[string]interface{}
	if err := json.NewDecoder(guidanceResp.Body).Decode(&guidanceData); err != nil {
		return s.error(err.Error()), nil
	}

	guidanceData, err = s.retryWithValidToken(account, password, "https://jw.hnit.edu.cn/njwhd/student/guidanceTeaching?token="+token, func(url string) (map[string]interface{}, error) {
		resp, err := httpClient.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var data map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&data)
		return data, err
	})
	if err != nil {
		return s.error(err.Error()), nil
	}

	if guidanceData["code"].(string) != "1" {
		return s.error(guidanceData["Msg"].(string)), nil
	}

	return &model.CourseResponse{Success: true, Data: guidanceData["data"]}, nil
}
