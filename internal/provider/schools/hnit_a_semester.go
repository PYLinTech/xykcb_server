package schools

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"xykcb_server/internal/cache"
	"xykcb_server/internal/config"
)

const defaultSemesterConfigTTL = 30 * 24 * time.Hour

type hnitASemesterCacheState struct {
	sync.Mutex
	expiresAt time.Time
	signature string
	semesters map[string]config.SemesterConfig
}

var hnitASemesterCache hnitASemesterCacheState

func semesterSortKey(semesterID string) int {
	key, _ := strconv.Atoi(strings.ReplaceAll(semesterID, "-", ""))
	return key
}

func cloneSemesterConfigs(semesters map[string]config.SemesterConfig) map[string]config.SemesterConfig {
	cloned := make(map[string]config.SemesterConfig, len(semesters))
	for k, v := range semesters {
		cloned[k] = v
	}
	return cloned
}

func semesterConfigSignature(schoolCfg *config.SchoolSemesters) string {
	data, _ := json.Marshal(struct {
		TTL  int                         `json:"ttl"`
		From []config.SemesterConfigFrom `json:"from"`
	}{TTL: schoolCfg.SemesterConfigTTL, From: schoolCfg.SemesterConfigFrom})
	return string(data)
}

func semesterConfigTTL(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultSemesterConfigTTL
	}
	return time.Duration(seconds) * time.Second
}

func selectSemesterConfigFrom(semesterID string, rules []config.SemesterConfigFrom) *config.SemesterConfigFrom {
	if len(rules) == 0 {
		return nil
	}

	sortedRules := append([]config.SemesterConfigFrom(nil), rules...)
	sort.Slice(sortedRules, func(i, j int) bool {
		return semesterSortKey(sortedRules[i].From) < semesterSortKey(sortedRules[j].From)
	})

	semesterKey := semesterSortKey(semesterID)
	selected := -1
	for i, rule := range sortedRules {
		if semesterSortKey(rule.From) <= semesterKey {
			selected = i
			continue
		}
		break
	}
	if selected == -1 {
		return nil
	}
	return &sortedRules[selected]
}

func parseSemesterStartFromWeek2(data map[string]interface{}) (string, error) {
	semesterData, ok := data["data"].([]interface{})
	if !ok || len(semesterData) == 0 {
		return "", fmt.Errorf("missing week 2 data")
	}

	firstWeek, ok := semesterData[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid week 2 data")
	}
	dates, ok := firstWeek["date"].([]interface{})
	if !ok {
		return "", fmt.Errorf("missing week 2 dates")
	}

	for _, item := range dates {
		dateItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if xqid, ok := dateItem["xqid"].(float64); !ok || int(xqid) != 1 {
			continue
		}

		monday, err := time.Parse("2006-01-02", safeString(dateItem["mxrq"], ""))
		if err != nil {
			return "", err
		}
		return monday.AddDate(0, 0, -7).Format("2006-01-02"), nil
	}

	return "", fmt.Errorf("missing week 2 monday")
}

func fetchHNITASemesterIDs(token string) ([]string, error) {
	reqURL := "https://jw.hnit.edu.cn/njwhd/getXnxqList?" + url.Values{"token": {token}}.Encode()
	resp, err := schoolRawClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	var payload interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if errorPayload, ok := payload.(map[string]interface{}); ok {
		return nil, fmt.Errorf("%s", safeString(errorPayload["Msg"], "semester list failed"))
	}

	items, ok := payload.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid semester list")
	}

	semesterIDs := make([]string, 0, len(items))
	for _, item := range items {
		semester, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		semesterID := safeString(semester["xnxq01id"], "")
		if semesterID != "" {
			semesterIDs = append(semesterIDs, semesterID)
		}
	}
	return semesterIDs, nil
}

func (s *HnitA) getSemesterIDs(account, password, token string) ([]string, error) {
	semesterIDs, err := fetchHNITASemesterIDs(token)
	if err == nil {
		return semesterIDs, nil
	}

	cache.InvalidateToken(s.GetProviderKey(), account)
	newToken, tokenErr := s.getToken(account, password)
	if tokenErr != nil {
		return nil, tokenErr
	}
	return fetchHNITASemesterIDs(newToken)
}

func (s *HnitA) getSemesterStart(account, password, token, semesterID string) (string, error) {
	curriculumData, err := s.retryWithValidToken(account, password, "/student/curriculum?token="+token+"&xnxq01id="+semesterID+"&week=2", func(path string) (map[string]interface{}, error) {
		return schoolClient.Get(path)
	})
	if err != nil {
		return "", err
	}
	if safeString(curriculumData["code"], "") != "1" {
		return "", fmt.Errorf("%s", safeString(curriculumData["Msg"], "curriculum failed"))
	}
	return parseSemesterStartFromWeek2(curriculumData)
}

func (s *HnitA) getSemesterConfigs(account, password, token string, schoolCfg *config.SchoolSemesters) map[string]config.SemesterConfig {
	if len(schoolCfg.SemesterConfigFrom) == 0 {
		return cloneSemesterConfigs(schoolCfg.Semesters)
	}

	signature := semesterConfigSignature(schoolCfg)
	hnitASemesterCache.Lock()
	defer hnitASemesterCache.Unlock()

	if time.Now().Before(hnitASemesterCache.expiresAt) && hnitASemesterCache.signature == signature && hnitASemesterCache.semesters != nil {
		return cloneSemesterConfigs(hnitASemesterCache.semesters)
	}

	semesterIDs, err := s.getSemesterIDs(account, password, token)
	if err != nil {
		log.Printf("hnit_a semester config: failed to fetch semester list: %v", err)
		return cloneSemesterConfigs(hnitASemesterCache.semesters)
	}

	semesters := make(map[string]config.SemesterConfig, len(semesterIDs))
	allStartsResolved := true
	for _, semesterID := range semesterIDs {
		rule := selectSemesterConfigFrom(semesterID, schoolCfg.SemesterConfigFrom)
		if rule == nil {
			continue
		}

		semesterStart, err := s.getSemesterStart(account, password, token, semesterID)
		if err != nil {
			log.Printf("hnit_a semester config: failed to calculate semesterStart for %s: %v", semesterID, err)
			allStartsResolved = false
			if cached, ok := hnitASemesterCache.semesters[semesterID]; ok {
				semesterStart = cached.SemesterStart
			}
		}

		semesters[semesterID] = config.SemesterConfig{
			SemesterStart:     semesterStart,
			TotalWeeks:        rule.TotalWeeks,
			TimeSlots:         rule.TimeSlots,
			MergeableSections: rule.MergeableSections,
		}
	}

	if len(semesters) == 0 {
		return cloneSemesterConfigs(hnitASemesterCache.semesters)
	}

	if allStartsResolved {
		hnitASemesterCache.signature = signature
		hnitASemesterCache.semesters = semesters
		hnitASemesterCache.expiresAt = time.Now().Add(semesterConfigTTL(schoolCfg.SemesterConfigTTL))
	}
	return cloneSemesterConfigs(semesters)
}
