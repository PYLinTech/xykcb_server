package schools

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

func (s *HnitA) GetSchoolId() string   { return "1" }
func (s *HnitA) GetNameZhcn() string   { return "湖南工学院（移动端）" }
func (s *HnitA) GetNameEn() string     { return "Hunan Institute Of Technology (Mobile)" }

func (s *HnitA) GetSchoolConfig() *config.SchoolSemesters {
	return config.GetSchoolConfigById("1")
}

func (s *HnitA) error(msg string) *model.CourseResponse {
	if strings.Contains(msg, "密码错误") {
		return &model.CourseResponse{Success: false, MsgZhcn: "账号或密码错误", MsgEn: "Invalid account or password"}
	}
	return &model.CourseResponse{Success: false, MsgZhcn: "登录失败", MsgEn: "Login failed"}
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

func (s *HnitA) Login(account, password string) (*model.CourseResponse, error) {
	resp, err := httpClient.Post("https://jw.hnit.edu.cn/njwhd/login?userNo="+account+"&pwd="+s.encryptPassword(password), "", nil)
	if err != nil {
		return s.error(err.Error()), nil
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return s.error(err.Error()), nil
	}

	if data["code"].(string) != "1" {
		return s.error(data["Msg"].(string)), nil
	}

	token := data["data"].(map[string]interface{})["token"].(string)
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

			curriculumResp, err := httpClient.Get("https://jw.hnit.edu.cn/njwhd/student/curriculum?token=" + token + "&xnxq01id=" + semesterID)
			if err != nil {
				return
			}
			defer curriculumResp.Body.Close()

			var curriculumData map[string]interface{}
			if err := json.NewDecoder(curriculumResp.Body).Decode(&curriculumData); err != nil {
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
				"semesterStart": semesterConfig.SemesterStart,
				"totalWeeks":    semesterConfig.TotalWeeks,
				"timeSlots":     semesterConfig.TimeSlots,
				"courses":       courses,
			}
			mu.Unlock()
		}(semesterID, semesterConfig)
	}

	wg.Wait()

	return &model.CourseResponse{Success: true, Data: result}, nil
}
