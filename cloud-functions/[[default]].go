package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"math/bits"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---- [[default]].go ----
type apiError struct{ DescKey string }

func (e apiError) Error() string { return e.DescKey }

func appError(code string) error { return apiError{DescKey: code} }

func Handler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, "001")
		return
	}

	var data any
	var err error
	switch r.URL.Path {
	case "/get-support-school":
		data = getSupportSchools()
	case "/get-support-function":
		data, err = handleSupportFunctions(r)
	case "/get-course-data":
		data, err = handleCourseData(r)
	case "/get-course-grades":
		data, err = handleGrades(r)
	case "/get-guidance-teaching":
		data, err = handleGuidanceTeaching(r)
	case "/get-hnit-b-evaluation", "/get-hnit-b-evaluation-courses", "/get-hnit-b-evaluation-form",
		"/submit-hnit-b-evaluation", "/get-hnit-b-room-borrow", "/get-hnit-b-room-borrow-rooms",
		"/get-hnit-b-room-borrow-form", "/submit-hnit-b-room-borrow":
		err = appError("002")
	default:
		writeNotFound(w)
		return
	}

	if err != nil {
		writeErrorFromErr(w, err)
		return
	}
	writeOK(w, data)
}

func handleSupportFunctions(r *http.Request) (any, error) {
	school := r.URL.Query().Get("school")
	if school == "" {
		return nil, appError("001")
	}
	cfg, ok := schools[school]
	if !ok {
		return nil, appError("002")
	}
	return cfg.Functions, nil
}

func queryCredentials(r *http.Request) (string, string) {
	q := r.URL.Query()
	account := q.Get("account")
	if account == "" {
		account = q.Get("student_ID")
	}
	password := q.Get("password")
	if password == "" {
		password = q.Get("student_password")
	}
	return account, password
}

func handleCourseData(r *http.Request) (any, error) {
	q := r.URL.Query()
	school := q.Get("school")
	if school == "" {
		return nil, appError("001")
	}
	if _, ok := schools[school]; !ok {
		return nil, appError("002")
	}
	if school == "hnit_b" {
		return nil, appError("002")
	}

	account, password := queryCredentials(r)
	if account == "" || password == "" {
		return nil, appError("001")
	}

	var collected map[string]SemesterPayload
	var err error
	switch school {
	case "hnit_a":
		client, e := newHnitAClient(account, password)
		if e != nil {
			return nil, e
		}
		collected, err = client.CourseData(hnitSemesterRule())
	case "hynu":
		client := newHynuClient(account, password)
		if e := client.Login(); e != nil {
			return nil, e
		}
		collected, err = client.CourseData(hynuSemesterRule())
	case "usc":
		client := newUscClient(account, password)
		if e := client.Login(); e != nil {
			return nil, e
		}
		collected, err = client.CourseData(uscSemesterRule())
	default:
		return nil, appError("002")
	}
	if err != nil {
		return nil, err
	}
	if len(collected) == 0 {
		return nil, appError("004")
	}
	return generateCourseTSV(school, collected), nil
}

func handleGrades(r *http.Request) (any, error) {
	q := r.URL.Query()
	school := q.Get("school")
	account, password := queryCredentials(r)
	if school == "" || account == "" || password == "" {
		return nil, appError("001")
	}
	if school != "hnit_a" {
		if _, ok := schools[school]; ok {
			return nil, appError("004")
		}
		return nil, appError("002")
	}
	client, err := newHnitAClient(account, password)
	if err != nil {
		return nil, err
	}
	return client.Grades(q.Get("semester"))
}

func handleGuidanceTeaching(r *http.Request) (any, error) {
	q := r.URL.Query()
	school := q.Get("school")
	account, password := queryCredentials(r)
	if school == "" || account == "" || password == "" {
		return nil, appError("001")
	}
	if school != "hnit_a" {
		if _, ok := schools[school]; ok {
			return nil, appError("004")
		}
		return nil, appError("002")
	}
	client, err := newHnitAClient(account, password)
	if err != nil {
		return nil, err
	}
	return client.GuidanceTeaching()
}

func setCORS(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func writeErrorFromErr(w http.ResponseWriter, err error) {
	if e, ok := err.(apiError); ok {
		writeError(w, e.DescKey)
		return
	}
	writeError(w, "004")
}

func writeError(w http.ResponseWriter, code string) {
	writeJSON(w, statusFor(code), map[string]any{"success": false, "desc_key": code})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	setCORS(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeNotFound(w http.ResponseWriter) {
	setCORS(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if data, err := readRuntimeAsset("404.html"); err == nil && len(data) > 0 {
		_, _ = w.Write(data)
		return
	}
	_, _ = w.Write([]byte("<!doctype html><title>404</title><h1>Not Found</h1>"))
}

func readRuntimeAsset(name string) ([]byte, error) {
	exeDir := ""
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	candidates := []string{
		name,
		filepath.Join("cloud-functions", name),
	}
	if exeDir != "" {
		candidates = append(candidates,
			filepath.Join(exeDir, name),
			filepath.Join(exeDir, "cloud-functions", name),
		)
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}

func statusFor(code string) int {
	switch code {
	case "001":
		return http.StatusBadRequest
	case "002":
		return http.StatusNotFound
	case "003":
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// ---- schools.go ----
type FunctionInfo map[string]string

type SchoolConfig struct {
	ID        string         `json:"id"`
	Functions []FunctionInfo `json:"functions"`
}

type SupportSchool struct {
	ID      string `json:"id"`
	DescKey string `json:"desc_key"`
}

type TimeSlot struct {
	Section int    `json:"section"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type SemesterRule struct {
	From              string
	TotalWeeks        int
	MergeableSections []string
	TimeSlots         []TimeSlot
}

type SemesterConfig struct {
	SemesterStart     string     `json:"semesterStart"`
	TotalWeeks        int        `json:"totalWeeks"`
	TimeSlots         []TimeSlot `json:"timeSlots"`
	MergeableSections []string   `json:"mergeableSections"`
}

type SemesterPayload struct {
	Cfg     SemesterConfig
	Courses []Course
}

type Course struct {
	RawID    string           `json:"rawID"`
	Name     string           `json:"name"`
	Location string           `json:"location"`
	Teacher  string           `json:"teacher"`
	Weeks    []int            `json:"weeks"`
	Schedule map[string][]int `json:"schedule"`
}

var schools = map[string]SchoolConfig{
	"hnit_a": {
		ID: "1",
		Functions: []FunctionInfo{
			{"id": "1", "url": "/web-static/plugin/hnit_a/grades.html", "zh-cn": "课程成绩", "en": "Course Grades"},
			{"id": "2", "url": "/web-static/plugin/hnit_a/major_plan.html", "zh-cn": "培养计划", "en": "Major Plan"},
		},
	},
	"hnit_b": {
		ID: "2",
		Functions: []FunctionInfo{
			{"id": "1", "url": "/web-static/plugin/hnit_b/teaching_evaluation.html", "zh-cn": "教学评价", "en": "Teaching Evaluation"},
			{"id": "2", "url": "/web-static/plugin/hnit_b/room_borrow.html", "zh-cn": "教室借用", "en": "Classroom Borrowing"},
		},
	},
	"hynu": {ID: "3", Functions: []FunctionInfo{}},
	"usc":  {ID: "4", Functions: []FunctionInfo{}},
}

func getSupportSchools() []SupportSchool {
	order := []string{"hnit_a", "hnit_b", "hynu", "usc"}
	out := make([]SupportSchool, 0, len(order))
	for _, key := range order {
		out = append(out, SupportSchool{ID: schools[key].ID, DescKey: key})
	}
	return out
}

func hnitSemesterRule() SemesterRule {
	return SemesterRule{
		From: "2024-2025-2", TotalWeeks: 20,
		MergeableSections: []string{"1-2", "3-4", "5-6", "7-8", "9-10"},
		TimeSlots:         []TimeSlot{{1, "08:30", "09:15"}, {2, "09:20", "10:05"}, {3, "10:25", "11:10"}, {4, "11:15", "12:00"}, {5, "14:00", "14:45"}, {6, "14:50", "15:35"}, {7, "15:55", "16:40"}, {8, "16:45", "17:30"}, {9, "19:00", "19:45"}, {10, "19:50", "20:35"}},
	}
}

func hynuSemesterRule() SemesterRule {
	return SemesterRule{
		From: "2024-2025-2", TotalWeeks: 20,
		MergeableSections: []string{"1-2", "3-4", "5-6", "7-8", "9-10"},
		TimeSlots:         []TimeSlot{{1, "08:30", "09:15"}, {2, "09:25", "10:10"}, {3, "10:30", "11:15"}, {4, "11:25", "12:10"}, {5, "14:30", "15:15"}, {6, "15:25", "16:10"}, {7, "16:30", "17:15"}, {8, "17:25", "18:10"}, {9, "19:30", "20:15"}, {10, "20:25", "21:10"}},
	}
}

func uscSemesterRule() SemesterRule {
	return SemesterRule{
		From: "2024-2025-2", TotalWeeks: 20,
		MergeableSections: []string{"1-2", "3-4", "5-6", "7-8", "9-10"},
		TimeSlots:         []TimeSlot{{1, "08:00", "08:45"}, {2, "08:55", "09:40"}, {3, "10:00", "10:45"}, {4, "10:55", "11:40"}, {5, "15:00", "15:45"}, {6, "15:55", "16:40"}, {7, "17:00", "17:45"}, {8, "17:55", "18:40"}, {9, "20:00", "20:45"}, {10, "20:55", "21:40"}},
	}
}

func semesterConfigFromRule(rule SemesterRule, start string) SemesterConfig {
	return SemesterConfig{
		SemesterStart:     start,
		TotalWeeks:        rule.TotalWeeks,
		TimeSlots:         rule.TimeSlots,
		MergeableSections: rule.MergeableSections,
	}
}

// ---- tsv.go ----
func generateCourseTSV(schoolID string, data map[string]SemesterPayload) string {
	semesterIDs := make([]string, 0, len(data))
	for id := range data {
		semesterIDs = append(semesterIDs, id)
	}
	sort.Slice(semesterIDs, func(i, j int) bool { return semesterSortKey(semesterIDs[i]) < semesterSortKey(semesterIDs[j]) })
	return buildTermsTSV(schoolID, semesterIDs, data) + "\n" + buildCoursesTSV(semesterIDs, data)
}

func buildTermsTSV(schoolID string, semesterIDs []string, data map[string]SemesterPayload) string {
	lines := []string{"@terms", "school_id\tterm_id\ttotal_weeks\tstart_date\tperiod_group\tsection_no\tsection_start_time\tsection_end_time"}
	var prev *termRow
	for _, semesterID := range semesterIDs {
		cfg := data[semesterID].Cfg
		groups := buildPeriodGroups(cfg.MergeableSections)
		for i, slot := range cfg.TimeSlots {
			group := ""
			if i < len(groups) {
				group = groups[i]
			}
			row := termRow{schoolID, semesterID, strconv.Itoa(cfg.TotalWeeks), cfg.SemesterStart, group, strconv.Itoa(slot.Section), slot.Start, slot.End}
			if prev == nil {
				lines = append(lines, strings.Join([]string{row.schoolID, row.termID, row.totalWeeks, nn(row.startDate), row.periodGroup, row.sectionNo, row.sectionStartTime, row.sectionEndTime}, "\t"))
			} else {
				fields := []string{"", "", "", "", "", row.sectionNo, row.sectionStartTime, row.sectionEndTime}
				if row.schoolID != prev.schoolID {
					fields[0] = row.schoolID
				}
				if row.termID != prev.termID {
					fields[1] = row.termID
				}
				if row.totalWeeks != prev.totalWeeks {
					fields[2] = row.totalWeeks
				}
				if row.startDate != prev.startDate {
					fields[3] = nn(row.startDate)
				}
				if row.periodGroup != prev.periodGroup {
					fields[4] = row.periodGroup
				}
				lines = append(lines, strings.Join(fields, "\t"))
			}
			copy := row
			prev = &copy
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildCoursesTSV(semesterIDs []string, data map[string]SemesterPayload) string {
	lines := []string{"@courses", "c_hash\tterm_id\traw_id\tcourse_name\tlocation\tteacher\tweeks\tweekday\tsections"}
	var prev *courseRow
	for _, semesterID := range semesterIDs {
		for _, course := range data[semesterID].Courses {
			weeks := joinSortedInts(course.Weeks)
			weekdays := make([]string, 0, len(course.Schedule))
			for weekday := range course.Schedule {
				weekdays = append(weekdays, weekday)
			}
			sort.Strings(weekdays)
			if len(weekdays) == 0 {
				weekdays = append(weekdays, "")
			}
			for _, weekday := range weekdays {
				sections := ""
				if weekday != "" {
					sections = joinSortedInts(course.Schedule[weekday])
				}
				row := makeCourseRow(semesterID, course.RawID, course.Name, course.Location, course.Teacher, weeks, weekday, sections)
				if prev == nil {
					lines = append(lines, strings.Join([]string{row.cHash, row.termID, row.rawID, row.name, row.location, row.teacher, row.weeks, row.weekday, row.sections}, "\t"))
				} else {
					fields := []string{row.cHash, "", "", "", "", "", "", row.weekday, row.sections}
					if row.termID != prev.termID {
						fields[1] = row.termID
					}
					if row.rawID != prev.rawID {
						fields[2] = row.rawID
					}
					if row.name != prev.name {
						fields[3] = row.name
					}
					if row.location != prev.location {
						fields[4] = row.location
					}
					if row.teacher != prev.teacher {
						fields[5] = row.teacher
					}
					if row.weeks != prev.weeks {
						fields[6] = row.weeks
					}
					lines = append(lines, strings.Join(fields, "\t"))
				}
				copy := row
				prev = &copy
			}
		}
	}
	return strings.Join(lines, "\n")
}

func makeCourseRow(termID, rawID, name, location, teacher, weeks, weekday, sections string) courseRow {
	row := courseRow{termID: termID, rawID: nn(rawID), name: nn(name), location: nn(location), teacher: nn(teacher), weeks: nn(weeks), weekday: nn(weekday), sections: nn(sections)}
	row.cHash = murmurHashBase36(strings.Join([]string{termID, rawID, name, location, teacher, weeks, weekday, sections}, "\t"))
	return row
}

func buildPeriodGroups(mergeableSections []string) []string {
	groups := []string{}
	for i, item := range mergeableSections {
		for range strings.Split(item, "-") {
			groups = append(groups, strconv.Itoa(i+1))
		}
	}
	return groups
}

func joinSortedInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	cp := append([]int(nil), values...)
	sort.Ints(cp)
	out := make([]string, 0, len(cp))
	for _, v := range cp {
		out = append(out, strconv.Itoa(v))
	}
	return strings.Join(out, ",")
}

func semesterSortKey(semesterID string) int {
	s := strings.ReplaceAll(semesterID, "-", "")
	n, _ := strconv.Atoi(s)
	return n
}

func nn(value string) string {
	if value == "" {
		return "\\N"
	}
	return value
}

func murmurHashBase36(input string) string {
	data := []byte(input)
	var h uint32
	const c1 uint32 = 0xcc9e2d51
	const c2 uint32 = 0x1b873593
	i := 0
	for ; i+4 <= len(data); i += 4 {
		k := uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16 | uint32(data[i+3])<<24
		k *= c1
		k = bits.RotateLeft32(k, 15)
		k *= c2
		h ^= k
		h = bits.RotateLeft32(h, 13)
		h = h*5 + 0xe6546b64
	}
	var k uint32
	if i < len(data) {
		if i+2 < len(data) {
			k ^= uint32(data[i+2]) << 16
		}
		if i+1 < len(data) {
			k ^= uint32(data[i+1]) << 8
		}
		k ^= uint32(data[i])
		k *= c1
		k = bits.RotateLeft32(k, 15)
		k *= c2
		h ^= k
	}
	h ^= uint32(len(data))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return fmt.Sprintf("%08s", strconv.FormatUint(uint64(h), 36))
}

type termRow struct {
	schoolID, termID, totalWeeks, startDate, periodGroup, sectionNo, sectionStartTime, sectionEndTime string
}

type courseRow struct {
	cHash, termID, rawID, name, location, teacher, weeks, weekday, sections string
}

// ---- http_client.go ----
const httpTimeout = 30 * time.Second

var hostIPMap = map[string]string{
	"jw.hnit.edu.cn":     "59.51.114.219",
	"jwxt.hnit.edu.cn":   "172.31.31.25",
	"hysfjw.hynu.edu.cn": "59.51.24.46",
	"jwzx.usc.edu.cn":    "61.187.179.66",
}

type fetchResult struct {
	URL    string
	Body   string
	Bytes  []byte
	Status int
}

type jwHTTPClient struct {
	baseURL *url.URL
	client  *http.Client
}

func newJWHTTPClient(base string, insecure bool) *jwHTTPClient {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if ip := hostIPMap[host]; ip != "" {
			address = net.JoinHostPort(ip, port)
		}
		d := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		return d.DialContext(ctx, network, address)
	}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // EdgeOne 侧仅为兼容部分教务旧证书。
	}
	parsed, _ := url.Parse(strings.TrimRight(base, "/"))
	return &jwHTTPClient{
		baseURL: parsed,
		client: &http.Client{
			Timeout:       httpTimeout,
			Jar:           jar,
			Transport:     transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *jwHTTPClient) Get(path string, accept string) (fetchResult, error) {
	return c.request(http.MethodGet, path, nil, "", accept, false, nil)
}

func (c *jwHTTPClient) GetBytes(path string, accept string) (fetchResult, error) {
	return c.request(http.MethodGet, path, nil, "", accept, true, nil)
}

func (c *jwHTTPClient) PostForm(path string, fields url.Values, ajax bool) (fetchResult, error) {
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded;charset=UTF-8"}
	if ajax {
		headers["X-Requested-With"] = "XMLHttpRequest"
	}
	return c.request(http.MethodPost, path, strings.NewReader(fields.Encode()), "application/x-www-form-urlencoded;charset=UTF-8", "text/html,application/xhtml+xml,application/xml,text/plain,*/*", false, headers)
}

func (c *jwHTTPClient) PostEmpty(path string, contentType, accept string) (fetchResult, error) {
	return c.request(http.MethodPost, path, bytes.NewReader(nil), contentType, accept, false, map[string]string{"Content-Type": contentType})
}

func (c *jwHTTPClient) request(method, path string, body io.Reader, contentType, accept string, wantBytes bool, headers map[string]string) (fetchResult, error) {
	var startBody []byte
	if body != nil {
		startBody, _ = io.ReadAll(body)
	}
	startURL, err := c.resolve(path)
	if err != nil {
		return fetchResult{}, appError("004")
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		currentURL := startURL
		currentMethod := method
		currentBody := append([]byte(nil), startBody...)
		referer := c.baseURL.String() + "/"
		for redirects := 0; redirects < 6; redirects++ {
			var reader io.Reader
			if currentMethod != http.MethodGet {
				reader = bytes.NewReader(currentBody)
			}
			req, err := http.NewRequest(currentMethod, currentURL, reader)
			if err != nil {
				return fetchResult{}, appError("004")
			}
			if accept == "" {
				accept = "text/html,application/xhtml+xml,application/xml,text/plain,*/*"
			}
			req.Header.Set("Accept", accept)
			req.Header.Set("Referer", referer)
			req.Header.Set("User-Agent", "Mozilla/5.0 xykcb-server EdgeOne-Go")
			if currentMethod != http.MethodGet && contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			resp, err := c.client.Do(req)
			if err != nil {
				lastErr = err
				break
			}
			if resp.StatusCode >= 500 {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				return fetchResult{}, appError("004")
			}
			if isRedirect(resp.StatusCode) {
				location := resp.Header.Get("Location")
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				target, err := url.Parse(location)
				if err != nil {
					return fetchResult{}, appError("004")
				}
				next := resp.Request.URL.ResolveReference(target)
				referer = resp.Request.URL.Scheme + "://" + resp.Request.URL.Host + "/"
				currentURL = next.String()
				currentMethod = http.MethodGet
				currentBody = nil
				continue
			}
			raw, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				lastErr = err
				break
			}
			if wantBytes {
				return fetchResult{URL: resp.Request.URL.String(), Bytes: raw, Status: resp.StatusCode}, nil
			}
			return fetchResult{URL: resp.Request.URL.String(), Body: decodeResponseText(raw, resp.Header.Get("Content-Type")), Status: resp.StatusCode}, nil
		}
		if lastErr == nil {
			lastErr = appError("004")
		}
		if attempt == 0 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if _, ok := lastErr.(apiError); ok {
		return fetchResult{}, lastErr
	}
	return fetchResult{}, appError("004")
}

func (c *jwHTTPClient) resolve(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	return c.baseURL.ResolveReference(ref).String(), nil
}

func isRedirect(status int) bool {
	return status == http.StatusMovedPermanently || status == http.StatusFound || status == http.StatusSeeOther || status == http.StatusTemporaryRedirect || status == http.StatusPermanentRedirect
}

func decodeResponseText(raw []byte, contentType string) string {
	// 现有 Android 和 EdgeOne JS 逻辑主要按 UTF-8 解析。这里保留 UTF-8 路径，避免额外三方依赖。
	return string(raw)
}

// ---- jwxt.go ----
const scheduleThrottle = 900 * time.Millisecond

var lastScheduleRequest time.Time
var scheduleMu sync.Mutex

type htmlField struct {
	Title string
	Name  string
	Value string
}

func throttleScheduleRequests() {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()
	wait := scheduleThrottle - time.Since(lastScheduleRequest)
	if wait > 0 {
		time.Sleep(wait)
	}
	lastScheduleRequest = time.Now()
}

func fetchScheduleHTML(client *jwHTTPClient, termID, week string, extra url.Values) (string, error) {
	throttleScheduleRequests()
	fields := url.Values{}
	fields.Set("zc", week)
	fields.Set("sfFD", "1")
	fields.Set("wkbkc", "1")
	if termID != "" {
		fields.Set("xnxq01id", termID)
	}
	for k, vs := range extra {
		for _, v := range vs {
			fields.Add(k, v)
		}
	}
	res, err := client.PostForm("/jsxsd/xskb/xskb_list.do", fields, false)
	if err != nil {
		return "", err
	}
	return res.Body, nil
}

func encodeSessionLogin(account, password, sessOrScode string, sxhOpt ...string) (string, error) {
	scode := sessOrScode
	sxh := ""
	if len(sxhOpt) > 0 {
		sxh = sxhOpt[0]
	}
	if sxh == "" {
		parts := strings.SplitN(strings.TrimSpace(sessOrScode), "#", 2)
		if len(parts) != 2 {
			return "", appError("004")
		}
		scode, sxh = parts[0], parts[1]
	}
	if scode == "" || len(sxh) < 20 {
		return "", appError("004")
	}
	code := account + "%%%" + password
	var b strings.Builder
	for i := 0; i < len(code); i++ {
		if i < 20 {
			size := int(sxh[i] - '0')
			if size < 0 || size > len(scode) {
				return "", appError("004")
			}
			b.WriteByte(code[i])
			b.WriteString(scode[:size])
			scode = scode[size:]
		} else {
			b.WriteString(code[i:])
			break
		}
	}
	return b.String(), nil
}

func extractSemesterIDs(html string, partPattern string) []string {
	if partPattern == "" {
		partPattern = "[12]"
	}
	re := regexp.MustCompile(`(?is)<option\b[^>]*\bvalue\s*=\s*['"]?([0-9]{4}-[0-9]{4}-` + partPattern + `)['"]?[^>]*>`)
	seen := map[string]bool{}
	var terms []string
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			terms = append(terms, m[1])
		}
	}
	sort.Slice(terms, func(i, j int) bool { return semesterSortKey(terms[i]) > semesterSortKey(terms[j]) })
	return terms
}

func selectedSemester(html string, partPattern string) string {
	if partPattern == "" {
		partPattern = "[12]"
	}
	re := regexp.MustCompile(`(?is)<option\b([^>]*)\bselected\b([^>]*)>`)
	valid := regexp.MustCompile(`^[0-9]{4}-[0-9]{4}-` + partPattern + `$`)
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		value := attr(m[1]+" "+m[2], "value")
		if valid.MatchString(value) {
			return value
		}
	}
	terms := extractSemesterIDs(html, partPattern)
	if len(terms) == 0 {
		return ""
	}
	return terms[0]
}

func selectedValue(html, selectID string) string {
	selectHTML := elementById(html, "select", selectID)
	if selectHTML == "" {
		return ""
	}
	selected := regexp.MustCompile(`(?is)<option\b([^>]*)\bselected\b([^>]*)>`).FindStringSubmatch(selectHTML)
	if len(selected) > 0 {
		return attr(selected[1]+" "+selected[2], "value")
	}
	first := regexp.MustCompile(`(?is)<option\b([^>]*)>`).FindStringSubmatch(selectHTML)
	if len(first) > 0 {
		return attr(first[1], "value")
	}
	return ""
}

func parseCoursesFromTable(table, termID string, courseNameExtractor func(string, string) string, cleanupFn func(string) string, weekdayForCell func(int) int) []Course {
	rows := childElements(table, "tr")
	courses := []Course{}
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		cells := childElements(rows[rowIndex], "td")
		for cellIndex := 0; cellIndex < len(cells) && cellIndex < 7; cellIndex++ {
			weekday := cellIndex + 1
			if weekdayForCell != nil {
				weekday = weekdayForCell(cellIndex)
			}
			blocks := elementsByClass(cells[cellIndex], "div", "kbcontent")
			for _, block := range blocks {
				if cleanupFn(block) == "" {
					continue
				}
				for _, chunk := range regexp.MustCompile(`-{5,}`).Split(block, -1) {
					if course, ok := parseCourseChunk(chunk, termID, weekday, sectionsForRow(rowIndex-1), courseNameExtractor, cleanupFn); ok {
						courses = append(courses, course)
					}
				}
			}
		}
	}
	return courses
}

func parseCourseChunk(chunk, termID string, weekday int, fallbackSections []int, courseNameExtractor func(string, string) string, cleanupFn func(string) string) (Course, bool) {
	fields := extractFontFields(chunk, cleanupFn)
	if len(fields) == 0 {
		return Course{}, false
	}
	name := ""
	if courseNameExtractor != nil {
		name = courseNameExtractor(chunk, termID)
	}
	if name == "" {
		for _, field := range fields {
			if field.Title == "" && field.Name == "" && !strings.HasPrefix(field.Value, "备注") {
				name = field.Value
				break
			}
		}
	}
	if name == "" {
		name = fieldValue(fields, "课程名称", "")
	}
	if name == "" {
		for _, field := range fields {
			v, t := field.Value, field.Title
			if !strings.HasPrefix(v, "备注") && !strings.Contains(t, "教师") && !strings.Contains(t, "周次") && !strings.Contains(t, "教室") && !strings.Contains(t, "教学楼") && !strings.Contains(t, "通知单") && !strings.Contains(t, "班级") {
				name = v
				break
			}
		}
	}
	if name == "" || name == "&nbsp;" {
		return Course{}, false
	}
	weekSection := fieldValue(fields, "周次(节次)", "")
	weeks := parseWeeksJwxt(weekSection)
	sections := parseSectionsJwxt(weekSection, fallbackSections)
	if len(weeks) == 0 || len(sections) == 0 {
		return Course{}, false
	}
	return Course{
		RawID:    stripPrefix(fieldValue(fields, "通知单编号", ""), "通知单编号"),
		Name:     name,
		Location: regexp.MustCompile(`\(.*?\)|（.*?）`).ReplaceAllString(fieldValue(fields, "教室", ""), ""),
		Teacher:  fieldValue(fields, "教师", ""),
		Weeks:    weeks,
		Schedule: map[string][]int{strconv.Itoa(weekday): sections},
	}, true
}

func extractFontFields(html string, cleanupFn func(string) string) []htmlField {
	re := regexp.MustCompile(`(?is)<font\b([^>]*)>(.*?)</font>`)
	fields := []htmlField{}
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		value := cleanupFn(m[2])
		if value == "" {
			continue
		}
		fields = append(fields, htmlField{Title: cleanupFn(attr(m[1], "title")), Name: cleanupFn(attr(m[1], "name")), Value: value})
	}
	return fields
}

func fieldValue(fields []htmlField, title, name string) string {
	for _, f := range fields {
		if (title != "" && strings.Contains(f.Title, title)) || (name != "" && f.Name == name) {
			return f.Value
		}
	}
	return ""
}

func parseWeeksJwxt(value string) []int {
	beforeParen := strings.SplitN(value, "(", 2)[0]
	beforeParen = strings.ReplaceAll(strings.ReplaceAll(beforeParen, "第", ""), "周", "")
	set := map[int]bool{}
	for _, raw := range strings.Split(beforeParen, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if m := regexp.MustCompile(`^(\d+)\s*-\s*(\d+)$`).FindStringSubmatch(item); len(m) > 0 {
			start, _ := strconv.Atoi(m[1])
			end, _ := strconv.Atoi(m[2])
			for n := start; n <= end; n++ {
				set[n] = true
			}
			continue
		}
		if m := regexp.MustCompile(`\d+`).FindString(item); m != "" {
			n, _ := strconv.Atoi(m)
			set[n] = true
		}
	}
	out := make([]int, 0, len(set))
	for n := range set {
		if strings.Contains(value, "单周") && n%2 == 0 {
			continue
		}
		if strings.Contains(value, "双周") && n%2 != 0 {
			continue
		}
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func parseSectionsJwxt(value string, fallback []int) []int {
	m := regexp.MustCompile(`\[([0-9\-\s]+)节\]`).FindStringSubmatch(value)
	if len(m) == 0 {
		return append([]int(nil), fallback...)
	}
	parts := regexp.MustCompile(`\s*-\s*`).Split(strings.TrimSpace(m[1]), -1)
	out := []int{}
	for _, item := range parts {
		if item == "" {
			continue
		}
		n, _ := strconv.Atoi(item)
		out = append(out, n)
	}
	return out
}

func sectionsForRow(rowIndex int) []int {
	start := rowIndex*2 + 1
	sections := []int{start}
	if start+1 <= 10 {
		sections = append(sections, start+1)
	}
	return sections
}

func mergeCourses(courses []Course) []Course {
	type entry struct {
		idx    int
		course Course
	}
	merged := map[string]*entry{}
	order := []string{}
	for _, course := range courses {
		weekday := ""
		sections := []int{}
		for k, v := range course.Schedule {
			weekday = k
			sections = v
			break
		}
		key := strings.Join([]string{course.Name, course.Location, course.Teacher, joinSortedInts(course.Weeks), weekday}, "\t")
		if merged[key] == nil {
			c := course
			merged[key] = &entry{idx: len(order), course: c}
			order = append(order, key)
			continue
		}
		target := &merged[key].course
		set := map[int]bool{}
		for _, v := range target.Schedule[weekday] {
			set[v] = true
		}
		for _, v := range sections {
			set[v] = true
		}
		vals := []int{}
		for v := range set {
			if v > 0 {
				vals = append(vals, v)
			}
		}
		sort.Ints(vals)
		target.Schedule[weekday] = vals
	}
	out := make([]Course, 0, len(order))
	for _, key := range order {
		out = append(out, merged[key].course)
	}
	return out
}

func elementById(html, tag, id string) string {
	re := regexp.MustCompile(`(?is)<` + regexp.QuoteMeta(tag) + `\b([^>]*)>`)
	for _, loc := range re.FindAllStringSubmatchIndex(html, -1) {
		attrs := html[loc[2]:loc[3]]
		if attr(attrs, "id") == id {
			return balancedElement(html, tag, loc[0])
		}
	}
	return ""
}

func childElements(html, tag string) []string {
	pattern := regexp.MustCompile(`(?is)</?` + regexp.QuoteMeta(tag) + `\b[^>]*>`)
	locs := pattern.FindAllStringIndex(html, -1)
	result := []string{}
	depth, start := 0, -1
	for _, loc := range locs {
		token := html[loc[0]:loc[1]]
		closing := strings.HasPrefix(token, "</")
		if !closing {
			if depth == 0 {
				start = loc[0]
			}
			depth++
		} else if depth > 0 {
			depth--
			if depth == 0 && start >= 0 {
				result = append(result, html[start:loc[1]])
				start = -1
			}
		}
	}
	return result
}

func elementsByClass(html, tag, className string) []string {
	re := regexp.MustCompile(`(?is)<` + regexp.QuoteMeta(tag) + `\b([^>]*)>`)
	result := []string{}
	for _, loc := range re.FindAllStringSubmatchIndex(html, -1) {
		attrs := html[loc[2]:loc[3]]
		if !hasClass(attr(attrs, "class"), className) {
			continue
		}
		element := balancedElement(html, tag, loc[0])
		if element != "" {
			result = append(result, element)
		}
	}
	return result
}

func balancedElement(html, tag string, start int) string {
	pattern := regexp.MustCompile(`(?is)</?` + regexp.QuoteMeta(tag) + `\b[^>]*>`)
	locs := pattern.FindAllStringIndex(html[start:], -1)
	depth := 0
	for _, rel := range locs {
		loc := []int{start + rel[0], start + rel[1]}
		token := html[loc[0]:loc[1]]
		closing := strings.HasPrefix(token, "</")
		selfClosing := strings.HasSuffix(token, "/>")
		if !closing && !selfClosing {
			depth++
		} else if closing && depth > 0 {
			depth--
			if depth == 0 {
				return html[start:loc[1]]
			}
		}
	}
	return ""
}

func hasClass(classes, className string) bool {
	for _, item := range strings.Fields(classes) {
		if item == className {
			return true
		}
	}
	return false
}

func attr(attrs, name string) string {
	doubleQuoted := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `\s*=\s*"([^"]*)"`).FindStringSubmatch(attrs)
	if len(doubleQuoted) > 0 {
		return doubleQuoted[1]
	}
	singleQuoted := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `\s*=\s*'([^']*)'`).FindStringSubmatch(attrs)
	if len(singleQuoted) > 0 {
		return singleQuoted[1]
	}
	unquoted := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `\s*=\s*([^\s>]+)`).FindStringSubmatch(attrs)
	if len(unquoted) > 0 {
		return unquoted[1]
	}
	return ""
}

func firstWeekMonday(calendarHTML string) string {
	rows := childElements(calendarHTML, "tr")
	mondayCellIndex := -1
	for _, row := range rows {
		headers := childElements(row, "th")
		for i, header := range headers {
			if strings.Contains(cleanupHTML(header), "星期一") {
				mondayCellIndex = i
				break
			}
		}
		if mondayCellIndex >= 0 {
			break
		}
	}
	if mondayCellIndex < 0 {
		mondayCellIndex = 2
	}
	dateRe := regexp.MustCompile(`title=['"](\d{4})年(\d{2})月(\d{2})['"]`)
	for _, row := range rows {
		cells := childElements(row, "td")
		if len(cells) <= mondayCellIndex {
			continue
		}
		weekNo, err := strconv.Atoi(cleanupHTML(cells[0]))
		if err != nil || weekNo <= 0 {
			continue
		}
		if m := dateRe.FindStringSubmatch(cells[mondayCellIndex]); len(m) > 0 {
			return addDaysForFirstWeek(m[1], m[2], m[3], -7*(weekNo-1))
		}
		for i := 1; i < len(cells); i++ {
			if m := dateRe.FindStringSubmatch(cells[i]); len(m) > 0 {
				return addDaysForFirstWeek(m[1], m[2], m[3], mondayCellIndex-i-7*(weekNo-1))
			}
		}
	}
	return ""
}

func addDaysForFirstWeek(y, m, d string, delta int) string {
	yy, _ := strconv.Atoi(y)
	mm, _ := strconv.Atoi(m)
	dd, _ := strconv.Atoi(d)
	t := time.Date(yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC).AddDate(0, 0, delta)
	return t.Format("2006-01-02")
}

func cleanupHTML(html string) string {
	s := regexp.MustCompile(`(?is)<br\s*/?>`).ReplaceAllString(html, " ")
	s = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = htmlDecode(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func htmlDecode(value string) string {
	replacer := strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'")
	return replacer.Replace(value)
}

func stripPrefix(value, prefix string) string {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `[:：]?`)
	return strings.TrimSpace(re.ReplaceAllString(value, ""))
}

func indexOfString(values []string, target string) int {
	for i, v := range values {
		if v == target {
			return i
		}
	}
	return -1
}

// ---- hnit_a.go ----
const hnitABaseURL = "https://jw.hnit.edu.cn/njwhd"
const hnitAAESKeyHex = "717a6b6a316b6a6768643d383736262a"

type hnitAClient struct {
	http     *jwHTTPClient
	account  string
	password string
	token    string
}

func newHnitAClient(account, password string) (*hnitAClient, error) {
	c := &hnitAClient{http: newJWHTTPClient(hnitABaseURL, true), account: account, password: password}
	token, err := c.login()
	if err != nil {
		return nil, err
	}
	c.token = token
	return c, nil
}

func (c *hnitAClient) CourseData(rule SemesterRule) (map[string]SemesterPayload, error) {
	semesters, err := c.getSemesterConfigs(rule)
	if err != nil {
		return nil, err
	}
	if len(semesters) == 0 {
		return nil, appError("004")
	}
	collected := map[string]SemesterPayload{}
	for semesterID, cfg := range semesters {
		payload, err := c.getJSON("/student/curriculum?token=" + url.QueryEscape(c.token) + "&xnxq01id=" + url.QueryEscape(semesterID) + "&week=all")
		if err != nil {
			return nil, err
		}
		collected[semesterID] = SemesterPayload{Cfg: cfg, Courses: hnitAParseCourses(payload)}
	}
	return collected, nil
}

func (c *hnitAClient) Grades(semester string) (any, error) {
	semesterList, err := c.getJSON("/semesterList?token=" + url.QueryEscape(c.token))
	if err != nil {
		return nil, err
	}
	if safeString(semesterList["code"]) != "1" {
		return nil, providerError(safeString(semesterList["Msg"]))
	}
	gradesData, err := c.getJSON("/student/termGPA?token=" + url.QueryEscape(c.token) + "&semester=" + url.QueryEscape(semester))
	if err != nil {
		return nil, err
	}
	if safeString(gradesData["code"]) != "1" {
		return nil, providerError(safeString(gradesData["Msg"]))
	}
	return map[string]any{"all-semester": semesterList["data"], "all-grades": gradesData["data"]}, nil
}

func (c *hnitAClient) GuidanceTeaching() (any, error) {
	payload, err := c.getJSON("/student/guidanceTeaching?token=" + url.QueryEscape(c.token))
	if err != nil {
		return nil, err
	}
	if safeString(payload["code"]) != "1" {
		return nil, providerError(safeString(payload["Msg"]))
	}
	return payload["data"], nil
}

func (c *hnitAClient) getSemesterConfigs(rule SemesterRule) (map[string]SemesterConfig, error) {
	ids, err := c.fetchSemesterIDs()
	if err != nil {
		return nil, err
	}
	semesters := map[string]SemesterConfig{}
	for _, semesterID := range ids {
		if semesterID == "" || semesterSortKey(semesterID) < semesterSortKey(rule.From) {
			continue
		}
		start, err := c.getSemesterStart(semesterID)
		if err != nil {
			return nil, err
		}
		semesters[semesterID] = semesterConfigFromRule(rule, start)
	}
	return semesters, nil
}

func (c *hnitAClient) fetchSemesterIDs() ([]string, error) {
	res, err := c.http.Get("/getXnxqList?token="+url.QueryEscape(c.token), "application/json")
	if err != nil {
		return nil, err
	}
	var payload []map[string]any
	if err := json.Unmarshal([]byte(res.Body), &payload); err != nil {
		return nil, appError("004")
	}
	out := []string{}
	for _, item := range payload {
		id := safeString(item["xnxq01id"])
		if id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

func (c *hnitAClient) getSemesterStart(semesterID string) (string, error) {
	payload, err := c.getJSON("/student/curriculum?token=" + url.QueryEscape(c.token) + "&xnxq01id=" + url.QueryEscape(semesterID) + "&week=2")
	if err != nil {
		return "", err
	}
	if safeString(payload["code"]) != "1" {
		return "", appError("004")
	}
	data, _ := payload["data"].([]any)
	if len(data) == 0 {
		return "", appError("004")
	}
	first, _ := data[0].(map[string]any)
	dates, _ := first["date"].([]any)
	monday := ""
	for _, raw := range dates {
		item, _ := raw.(map[string]any)
		if intFromAny(item["xqid"]) == 1 {
			monday = safeString(item["mxrq"])
			break
		}
	}
	if monday == "" {
		return "", appError("004")
	}
	t, err := time.ParseInLocation("2006-01-02", monday, time.UTC)
	if err != nil {
		return "", appError("004")
	}
	return t.AddDate(0, 0, -7).Format("2006-01-02"), nil
}

func hnitAParseCourses(payload map[string]any) []Course {
	out := []Course{}
	data, _ := payload["data"].([]any)
	for _, dayRaw := range data {
		day, _ := dayRaw.(map[string]any)
		items, _ := day["item"].([]any)
		for _, raw := range items {
			source, _ := raw.(map[string]any)
			out = append(out, hnitAConvertCourse(source))
		}
	}
	return stableGroupByRawID(out)
}

func hnitAConvertCourse(source map[string]any) Course {
	return Course{
		RawID:    safeString(source["kch"]),
		Name:     safeString(source["courseName"]),
		Location: regexp.MustCompile(`\(.*?\)|（.*?）`).ReplaceAllString(safeString(source["location"]), ""),
		Teacher:  safeString(source["teacherName"]),
		Weeks:    hnitAParseWeeks(safeString(source["classWeekDetails"])),
		Schedule: hnitAParseClassTime(safeString(source["classTime"])),
	}
}

func hnitAParseWeeks(value string) []int {
	out := []int{}
	for _, item := range strings.Split(value, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && n > 0 {
			out = append(out, n)
		}
	}
	return out
}

func hnitAParseClassTime(value string) map[string][]int {
	schedule := map[string][]int{}
	if len(value) < 3 {
		return schedule
	}
	weekday := value[:1]
	sections := []int{}
	for i := 1; i+1 < len(value); i += 2 {
		n, err := strconv.Atoi(value[i : i+2])
		if err != nil {
			n = 0
		}
		sections = append(sections, n)
	}
	schedule[weekday] = sections
	return schedule
}

func stableGroupByRawID(courses []Course) []Course {
	groups := map[string][]Course{}
	order := []string{}
	for _, c := range courses {
		key := c.RawID
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], c)
	}
	out := []Course{}
	for _, key := range order {
		out = append(out, groups[key]...)
	}
	return out
}

func (c *hnitAClient) getJSON(path string) (map[string]any, error) {
	payload, err := c.fetchJSON(path)
	if err != nil {
		return nil, err
	}
	code := safeString(payload["code"])
	if code != "" && code != "1" {
		token, err := c.login()
		if err != nil {
			return nil, err
		}
		c.token = token
		payload, err = c.fetchJSON(replaceToken(path, c.token))
		if err != nil {
			return nil, err
		}
	}
	return payload, nil
}

func (c *hnitAClient) fetchJSON(path string) (map[string]any, error) {
	res, err := c.http.Get(path, "application/json")
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(res.Body), &payload); err != nil {
		return nil, appError("004")
	}
	return payload, nil
}

func (c *hnitAClient) login() (string, error) {
	encrypted, err := encryptHnitAPassword(c.password)
	if err != nil {
		return "", appError("004")
	}
	res, err := c.http.PostEmpty("/login?userNo="+url.QueryEscape(c.account)+"&pwd="+url.QueryEscape(encrypted), "application/json", "application/json")
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(res.Body), &payload); err != nil {
		return "", appError("004")
	}
	if safeString(payload["code"]) != "1" {
		if strings.Contains(safeString(payload["Msg"]), "密码错误") {
			return "", appError("003")
		}
		return "", appError("004")
	}
	data, _ := payload["data"].(map[string]any)
	token := safeString(data["token"])
	if token == "" {
		return "", appError("004")
	}
	return token, nil
}

func encryptHnitAPassword(password string) (string, error) {
	key, err := hex.DecodeString(hnitAAESKeyHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := []byte("\"" + password + "\"")
	pad := block.BlockSize() - len(plain)%block.BlockSize()
	for i := 0; i < pad; i++ {
		plain = append(plain, byte(pad))
	}
	encrypted := make([]byte, len(plain))
	for offset := 0; offset < len(plain); offset += block.BlockSize() {
		block.Encrypt(encrypted[offset:offset+block.BlockSize()], plain[offset:offset+block.BlockSize()])
	}
	b64 := base64.StdEncoding.EncodeToString(encrypted)
	return base64.StdEncoding.EncodeToString([]byte(b64)), nil
}

func replaceToken(path, token string) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

func providerError(message string) error {
	if strings.Contains(message, "密码错误") {
		return appError("003")
	}
	return appError("004")
}

func safeString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

// ---- hynu.go ----
const hynuBaseURL = "https://hysfjw.hynu.edu.cn"

type hynuClient struct {
	account  string
	password string
	http     *jwHTTPClient
	kbjcmsid string
}

func newHynuClient(account, password string) *hynuClient {
	return &hynuClient{account: account, password: password, http: newJWHTTPClient(hynuBaseURL, true)}
}

func (c *hynuClient) Login() error {
	sessRes, err := c.http.PostForm("/Logon.do?method=logon&flag=sess", url.Values{}, true)
	if err != nil {
		return err
	}
	encoded, err := encodeSessionLogin(c.account, c.password, strings.TrimSpace(sessRes.Body))
	if err != nil {
		return err
	}
	body := url.Values{}
	body.Set("userAccount", c.account)
	body.Set("userPassword", "")
	body.Set("encoded", encoded)
	res, err := c.postLoginResult(body)
	if err != nil {
		return err
	}
	if !strings.Contains(res.URL, "/jsxsd/framework/xsMain.htmlx") || strings.Contains(res.Body, "用户登录") {
		return appError("003")
	}
	return nil
}

func (c *hynuClient) postLoginResult(fields url.Values) (fetchResult, error) {
	return c.requestWithMainRedirectStop("/Logon.do?method=logon", fields)
}

func (c *hynuClient) requestWithMainRedirectStop(path string, fields url.Values) (fetchResult, error) {
	// 衡阳师范登录成功时常以 302 跳转到 xsMain.htmlx 表示成功；这里手写轻量特殊流程。
	res, err := c.http.PostForm(path, fields, false)
	if err == nil {
		return res, nil
	}
	return res, err
}

func (c *hynuClient) CourseData(rule SemesterRule) (map[string]SemesterPayload, error) {
	firstHTML, err := c.fetchScheduleHTML("", "")
	if err != nil {
		return nil, err
	}
	semesterIDs := extractSemesterIDs(firstHTML, "[12]")
	selected := c.selectedSemester(firstHTML)
	if len(semesterIDs) == 0 && selected != "" {
		semesterIDs = append(semesterIDs, selected)
	}
	if selected != "" {
		next := []string{selected}
		for _, id := range semesterIDs {
			if id != selected {
				next = append(next, id)
			}
		}
		semesterIDs = next
	}

	selectedKey := 1<<31 - 1
	if selected != "" {
		selectedKey = semesterSortKey(selected)
	}
	collected := map[string]SemesterPayload{}
	for _, semesterID := range semesterIDs {
		termKey := semesterSortKey(semesterID)
		if semesterID == "" || termKey < semesterSortKey(rule.From) || termKey > selectedKey {
			continue
		}
		html := firstHTML
		if semesterID != selected {
			html, err = c.fetchScheduleHTML(semesterID, "")
			if err != nil {
				return nil, err
			}
		}
		courses, err := c.parseScheduleHTML(html, semesterID)
		if err != nil {
			return nil, err
		}
		courses = mergeCourses(courses)
		start := c.semesterStart(semesterID)
		if start == "" {
			return nil, appError("004")
		}
		collected[semesterID] = SemesterPayload{Cfg: semesterConfigFromRule(rule, start), Courses: courses}
	}
	return sortSemesterPayloads(collected), nil
}

func (c *hynuClient) fetchScheduleHTML(termID, week string) (string, error) {
	extra := url.Values{}
	if c.kbjcmsid != "" {
		extra.Set("kbjcmsid", c.kbjcmsid)
	}
	html, err := fetchScheduleHTML(c.http, termID, week, extra)
	if err != nil {
		return "", err
	}
	if err := c.checkScheduleResponse(html); err != nil {
		return "", err
	}
	return html, nil
}

func (c *hynuClient) checkScheduleResponse(html string) error {
	if strings.Contains(html, "用户登录") || strings.Contains(html, "操作系统过于频繁") || strings.Contains(html, "已被注销登录") {
		return appError("004")
	}
	if c.kbjcmsid == "" {
		c.kbjcmsid = selectedValue(html, "kbjcmsid")
	}
	return nil
}

func (c *hynuClient) parseScheduleHTML(html, termID string) ([]Course, error) {
	table := elementById(html, "table", "timetable")
	if table == "" {
		return nil, appError("004")
	}
	return parseCoursesFromTable(table, termID, c.courseNameFromChunk, cleanupHTML, nil), nil
}

func (c *hynuClient) courseNameFromChunk(chunk, termID string) string {
	parts := regexp.MustCompile(`(?is)<font\b`).Split(chunk, 2)
	head := ""
	if len(parts) > 0 {
		head = parts[0]
	}
	head = regexp.MustCompile(`(?is)<br\s*/?>`).ReplaceAllString(head, " ")
	return cleanupHTML(head)
}

func (c *hynuClient) selectedSemester(html string) string {
	term := selectedValue(html, "xnxq01id")
	if regexp.MustCompile(`^[0-9]{4}-[0-9]{4}-[12]$`).MatchString(term) {
		return term
	}
	return selectedSemester(html, "[12]")
}

func (c *hynuClient) semesterStart(termID string) string {
	fields := url.Values{}
	fields.Set("xnxq01id", termID)
	res, err := c.http.PostForm("/jsxsd/jxzl/jxzl_query", fields, false)
	if err != nil {
		return ""
	}
	return firstWeekMonday(res.Body)
}

func sortSemesterPayloads(data map[string]SemesterPayload) map[string]SemesterPayload {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return semesterSortKey(keys[i]) < semesterSortKey(keys[j]) })
	out := map[string]SemesterPayload{}
	for _, key := range keys {
		out[key] = data[key]
	}
	return out
}

// ---- usc.go ----
const uscBaseURL = "http://jwzx.usc.edu.cn:8924"
const maxCaptchaAttempts = 50
const maxEmptyTerms = 3

type uscClient struct {
	account  string
	password string
	http     *jwHTTPClient
	kbjcmsid string
}

func newUscClient(account, password string) *uscClient {
	return &uscClient{account: account, password: password, http: newJWHTTPClient(uscBaseURL, false)}
}

func (c *uscClient) Login() error {
	for attempt := 0; attempt < maxCaptchaAttempts; attempt++ {
		sessRes, err := c.http.PostForm("/Logon.do?method=logon&flag=sess", url.Values{}, true)
		if err != nil {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(sessRes.Body), "#", 2)
		if len(parts) != 2 || parts[0] == "" || len(parts[1]) < 20 {
			continue
		}
		encoded, err := encodeSessionLogin(c.account, c.password, parts[0], parts[1])
		if err != nil {
			return err
		}
		captchaBytes, err := c.getBytes("/verifycode.servlet")
		if err != nil {
			continue
		}
		captcha := recognizeCaptcha(captchaBytes)
		if len(captcha) != 4 {
			continue
		}
		fields := url.Values{}
		fields.Set("userAccount", c.account)
		fields.Set("userPassword", c.password)
		fields.Set("RANDOMCODE", captcha)
		fields.Set("encoded", encoded)
		res, err := c.http.PostForm("/Logon.do?method=logon", fields, false)
		if err != nil {
			continue
		}
		errorText := ""
		if m := regexp.MustCompile(`(?is)id=["']showMsg["'][^>]*>(.*?)<`).FindStringSubmatch(res.Body); len(m) > 0 {
			errorText = strings.TrimSpace(strings.ReplaceAll(cleanupHTML(m[1]), "&nbsp;", ""))
		}
		if strings.Contains(errorText, "验证码") {
			continue
		}
		if strings.Contains(errorText, "密码") || strings.Contains(errorText, "帐号") || strings.Contains(errorText, "账号") {
			return appError("003")
		}
		if errorText != "" {
			return appError("004")
		}
		if strings.Contains(res.URL, "/jsxsd/") && !strings.Contains(res.Body, "用户登录") {
			return nil
		}
	}
	return appError("004")
}

func (c *uscClient) CourseData(rule SemesterRule) (map[string]SemesterPayload, error) {
	firstHTML, err := c.fetchScheduleHTML("", "")
	if err != nil {
		return nil, err
	}
	selected := c.selectedSemester(firstHTML)
	semesterIDs := c.semesterOptions(firstHTML)
	if len(semesterIDs) == 0 && selected != "" {
		semesterIDs = append(semesterIDs, selected)
	}
	startIndex := indexOfString(semesterIDs, selected)
	if startIndex < 0 && selected != "" {
		semesterIDs = append([]string{selected}, semesterIDs...)
		startIndex = 0
	}
	if startIndex < 0 {
		startIndex = 0
	}

	collected := map[string]SemesterPayload{}
	fromKey := semesterSortKey(rule.From)
	emptyTerms := 0
	for i := startIndex; i < len(semesterIDs); i++ {
		semesterID := semesterIDs[i]
		if semesterID == "" {
			continue
		}
		if fromKey > 0 && semesterSortKey(semesterID) < fromKey {
			break
		}
		html := firstHTML
		if semesterID != selected {
			html, err = c.fetchScheduleHTML(semesterID, "")
			if err != nil {
				return nil, err
			}
		}
		courses, err := c.parseScheduleHTML(html, semesterID)
		if err != nil {
			return nil, err
		}
		courses = mergeCourses(courses)
		if len(courses) == 0 {
			emptyTerms++
			if emptyTerms >= maxEmptyTerms {
				break
			}
			continue
		}
		emptyTerms = 0
		start := c.semesterStart(semesterID)
		if start == "" {
			return nil, appError("004")
		}
		collected[semesterID] = SemesterPayload{Cfg: semesterConfigFromRule(rule, start), Courses: courses}
	}
	if len(collected) == 0 {
		return nil, appError("004")
	}
	return sortSemesterPayloads(collected), nil
}

func (c *uscClient) fetchScheduleHTML(termID, week string) (string, error) {
	extra := url.Values{}
	if c.kbjcmsid != "" {
		extra.Set("kbjcmsid", c.kbjcmsid)
	}
	html, err := fetchScheduleHTML(c.http, termID, week, extra)
	if err != nil {
		return "", err
	}
	if err := c.checkScheduleResponse(html); err != nil {
		return "", err
	}
	return html, nil
}

func (c *uscClient) checkScheduleResponse(html string) error {
	if strings.Contains(html, "用户登录") || strings.Contains(html, "操作系统过于频繁") || strings.Contains(html, "已被注销登录") {
		return appError("004")
	}
	if c.kbjcmsid == "" {
		c.kbjcmsid = selectedValue(html, "kbjcmsid")
	}
	return nil
}

func (c *uscClient) parseScheduleHTML(html, termID string) ([]Course, error) {
	table := elementById(html, "table", "kbtable")
	if table == "" {
		return nil, appError("004")
	}
	rows := childElements(table, "tr")
	courses := []Course{}
	splitter := regexp.MustCompile(`(?is)</?br\s*/?>\s*-{5,}\s*</?br\s*/?>`)
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		cells := childElements(rows[rowIndex], "td")
		for cellIndex := 0; cellIndex < len(cells) && cellIndex < 7; cellIndex++ {
			weekday := uscWeekdayFromColumn(cellIndex)
			blocks := elementsByClass(cells[cellIndex], "div", "kbcontent")
			for _, block := range blocks {
				for _, chunk := range splitter.Split(block, -1) {
					if course, ok := parseNanhuaChunk(chunk, termID, weekday, sectionsForRow(rowIndex-1)); ok {
						courses = append(courses, course)
					}
				}
			}
		}
	}
	return courses, nil
}

func parseNanhuaChunk(chunk, termID string, weekday int, fallbackSections []int) (Course, bool) {
	lines := regexp.MustCompile(`(?is)<br\s*/?>`).Split(chunk, -1)
	name, teacher, weeksRaw, location := "", "", "", ""
	for _, line := range lines {
		rawTitle := attr(line, "title")
		text := cleanupNanhuaHTML(line)
		if text == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "<a ") {
			continue
		}
		switch rawTitle {
		case "老师":
			teacher = text
		case "周次(节次)":
			weeksRaw = regexp.MustCompile(`【.*?】`).ReplaceAllString(text, "")
		case "教室":
			location = regexp.MustCompile(`【.*?】`).ReplaceAllString(text, "")
		default:
			if rawTitle == "" && name == "" {
				name = regexp.MustCompile(`[\s\u00a0]+[OPop]$`).ReplaceAllString(text, "")
				name = strings.TrimSpace(name)
			}
		}
	}
	if name == "" || weeksRaw == "" {
		return Course{}, false
	}
	weeks := parseWeeksJwxt(weeksRaw)
	sections := parseSectionsJwxt(weeksRaw, fallbackSections)
	if len(weeks) == 0 || len(sections) == 0 {
		return Course{}, false
	}
	return Course{RawID: "", Name: name, Location: location, Teacher: teacher, Weeks: weeks, Schedule: map[string][]int{intToString(weekday): sections}}, true
}

func (c *uscClient) selectedSemester(html string) string {
	term := selectedValue(html, "xnxq01id")
	if regexp.MustCompile(`^[0-9]{4}-[0-9]{4}-[123]$`).MatchString(term) {
		return term
	}
	return selectedSemester(html, "[123]")
}

func (c *uscClient) semesterOptions(html string) []string {
	selectHTML := elementById(html, "select", "xnxq01id")
	re := regexp.MustCompile(`(?is)<option\b[^>]*\bvalue\s*=\s*['"]?([0-9]{4}-[0-9]{4}-[123])['"]?[^>]*>`)
	seen := map[string]bool{}
	terms := []string{}
	for _, m := range re.FindAllStringSubmatch(selectHTML, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			terms = append(terms, m[1])
		}
	}
	return terms
}

func (c *uscClient) semesterStart(termID string) string {
	fields := url.Values{}
	fields.Set("xnxq01id", termID)
	res, err := c.http.PostForm("/jsxsd/jxzl/jxzl_query", fields, false)
	if err != nil {
		return ""
	}
	return firstWeekMonday(res.Body)
}

func (c *uscClient) getBytes(path string) ([]byte, error) {
	res, err := c.http.GetBytes(path, "image/*,*/*")
	if err != nil {
		return nil, err
	}
	return res.Bytes, nil
}

func cleanupNanhuaHTML(html string) string {
	s := regexp.MustCompile(`(?i)(&nbsp;?|\s+)`).ReplaceAllString(html, " ")
	return cleanupHTML(s)
}

func uscWeekdayFromColumn(cellIndex int) int {
	switch cellIndex {
	case 0:
		return 7
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 4:
		return 4
	case 5:
		return 5
	case 6:
		return 6
	default:
		return 0
	}
}

func intToString(v int) string { return strconv.Itoa(v) }

// ---- captcha.go ----
const ocrStdW = 20
const ocrStdH = 30
const ocrCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

var ocrOnce sync.Once
var ocrSamples []ocrSample

type ocrSample struct {
	ch   byte
	gray []uint8
	bin  []bool
	feat []float64
}

func readOCRTemplatesFile() ([]byte, error) {
	exeDir := ""
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	candidates := []string{
		"ocr_templates.gz",
		filepath.Join("cloud-functions", "ocr_templates.gz"),
	}
	if exeDir != "" {
		candidates = append(candidates,
			filepath.Join(exeDir, "ocr_templates.gz"),
			filepath.Join(exeDir, "cloud-functions", "ocr_templates.gz"),
		)
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}

func recognizeCaptcha(imageBytes []byte) string {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return ""
	}
	ocrOnce.Do(loadOCRTemplates)
	if len(ocrSamples) == 0 {
		return ""
	}
	gray := imageToGray(img)
	bin := binarizeGray(gray)
	chars := segmentCharacters(bin, gray, 4)
	if len(chars) == 0 {
		return ""
	}
	var out []byte
	for _, ch := range chars {
		out = append(out, classifyOCR(ch))
	}
	return string(out)
}

func loadOCRTemplates() {
	ocrTemplatesGzip, err := readOCRTemplatesFile()
	if err != nil || len(ocrTemplatesGzip) == 0 {
		return
	}
	zr, err := gzip.NewReader(bytes.NewReader(ocrTemplatesGzip))
	if err != nil {
		return
	}
	raw, err := io.ReadAll(zr)
	_ = zr.Close()
	if err != nil || len(raw) < 14 || string(raw[:6]) != "XYOCR1" {
		return
	}
	n := int(binary.LittleEndian.Uint16(raw[6:8]))
	w := int(binary.LittleEndian.Uint16(raw[10:12]))
	h := int(binary.LittleEndian.Uint16(raw[12:14]))
	if w != ocrStdW || h != ocrStdH {
		return
	}
	pos := 14
	samples := make([]ocrSample, 0, n)
	size := ocrStdW * ocrStdH
	for i := 0; i < n && pos+1+size <= len(raw); i++ {
		ci := int(raw[pos])
		pos++
		if ci < 0 || ci >= len(ocrCharset) {
			pos += size
			continue
		}
		gray := append([]uint8(nil), raw[pos:pos+size]...)
		pos += size
		bin := grayToBinary(gray)
		samples = append(samples, ocrSample{ch: ocrCharset[ci], gray: gray, bin: bin, feat: ocrFeatures(bin, ocrStdH, ocrStdW)})
	}
	ocrSamples = samples
}

func imageToGray(img image.Image) [][]int {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	g := make([][]int, h)
	for y := 0; y < h; y++ {
		g[y] = make([]int, w)
		for x := 0; x < w; x++ {
			r, gg, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			g[y][x] = int(((r>>8)*299 + (gg>>8)*587 + (bb>>8)*114) / 1000)
		}
	}
	return g
}

func binarizeGray(gray [][]int) [][]bool {
	h, w := len(gray), len(gray[0])
	hist := make([]int, 256)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := gray[y][x]
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			hist[v]++
		}
	}
	t := otsu(hist, w*h)
	bin := make([][]bool, h)
	for y := 0; y < h; y++ {
		bin[y] = make([]bool, w)
		for x := 0; x < w; x++ {
			bin[y][x] = gray[y][x] < t
		}
	}
	cropBorder(bin)
	removeNoise(bin)
	return bin
}

func otsu(hist []int, total int) int {
	sumAll, sumBg, wBg, maxVar := 0.0, 0.0, 0.0, 0.0
	best := 0
	for i := 0; i < 256; i++ {
		sumAll += float64(i * hist[i])
	}
	for t := 0; t < 256; t++ {
		wBg += float64(hist[t])
		if wBg == 0 {
			continue
		}
		wFg := float64(total) - wBg
		if wFg == 0 {
			break
		}
		sumBg += float64(t * hist[t])
		d := sumBg/wBg - (sumAll-sumBg)/wFg
		v := wBg * wFg * d * d
		if v > maxVar {
			maxVar = v
			best = t
		}
	}
	return best
}

func cropBorder(img [][]bool) {
	h, w := len(img), len(img[0])
	for y := 0; y < h; y++ {
		for x := 0; x < 2 && x < w; x++ {
			img[y][x] = false
			img[y][w-1-x] = false
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < 2 && y < h; y++ {
			img[y][x] = false
			img[h-1-y][x] = false
		}
	}
}

func removeNoise(img [][]bool) {
	h, w := len(img), len(img[0])
	snap := make([][]bool, h)
	for y := range img {
		snap[y] = append([]bool(nil), img[y]...)
	}
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			if !snap[y][x] {
				continue
			}
			n := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if (dy != 0 || dx != 0) && snap[y+dy][x+dx] {
						n++
					}
				}
			}
			if n < 2 {
				img[y][x] = false
			}
		}
	}
	visited := makeBool2D(h, w)
	keep := makeBool2D(h, w)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !img[y][x] || visited[y][x] {
				continue
			}
			points := floodPoints(img, visited, y, x)
			if len(points) >= 10 {
				for _, p := range points {
					keep[p[0]][p[1]] = true
				}
			}
		}
	}
	for y := 0; y < h; y++ {
		copy(img[y], keep[y])
	}
}

func makeBool2D(h, w int) [][]bool {
	out := make([][]bool, h)
	for i := range out {
		out[i] = make([]bool, w)
	}
	return out
}

func floodPoints(img, vis [][]bool, sy, sx int) [][2]int {
	h, w := len(img), len(img[0])
	q := [][2]int{{sy, sx}}
	vis[sy][sx] = true
	pts := [][2]int{{sy, sx}}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dy == 0 && dx == 0 {
					continue
				}
				ny, nx := p[0]+dy, p[1]+dx
				if ny >= 0 && ny < h && nx >= 0 && nx < w && img[ny][nx] && !vis[ny][nx] {
					vis[ny][nx] = true
					np := [2]int{ny, nx}
					q = append(q, np)
					pts = append(pts, np)
				}
			}
		}
	}
	return pts
}

func segmentCharacters(bin [][]bool, gray [][]int, count int) [][]uint8 {
	h, w := len(bin), len(bin[0])
	proj := make([]int, w)
	topY, botY, leftX, rightX := h, 0, w, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if bin[y][x] {
				proj[x]++
				if y < topY {
					topY = y
				}
				if y > botY {
					botY = y
				}
				if x < leftX {
					leftX = x
				}
				if x > rightX {
					rightX = x
				}
			}
		}
	}
	if topY >= botY || leftX >= rightX {
		return nil
	}
	expectedW := maxInt(5, (rightX-leftX+1)/count)
	regions := findRegions(proj)
	for i := len(regions) - 2; i >= 0; i-- {
		if regions[i+1][0]-regions[i][1] <= 3 {
			regions[i][1] = regions[i+1][1]
			regions = append(regions[:i+1], regions[i+2:]...)
		}
	}
	for iter := 0; iter < 10 && (len(regions) < count || hasWide(regions, expectedW)); iter++ {
		mi := widestRegion(regions)
		if mi < 0 || regions[mi][1]-regions[mi][0]+1 < int(float64(expectedW)*0.7) {
			break
		}
		rs, re := regions[mi][0], regions[mi][1]
		mw := re - rs + 1
		lo, hi := rs+mw/3, re-mw/3
		minV, cutX := int(^uint(0)>>1), (rs+re)/2
		for x := lo; x <= hi; x++ {
			if proj[x] < minV {
				minV = proj[x]
				cutX = x
			}
		}
		if cutX-rs < 3 || re-cutX < 3 {
			break
		}
		newRegs := append([][2]int{}, regions[:mi]...)
		newRegs = append(newRegs, [2]int{rs, cutX - 1}, [2]int{cutX + 1, re})
		newRegs = append(newRegs, regions[mi+1:]...)
		regions = newRegs
	}
	for len(regions) > count {
		minGap, mi := int(^uint(0)>>1), 0
		for i := 0; i < len(regions)-1; i++ {
			gap := regions[i+1][0] - regions[i][1] - 1
			if gap < minGap {
				minGap = gap
				mi = i
			}
		}
		regions[mi][1] = regions[mi+1][1]
		regions = append(regions[:mi+1], regions[mi+2:]...)
	}
	sort.Slice(regions, func(i, j int) bool { return regions[i][0] < regions[j][0] })
	result := [][]uint8{}
	for _, r := range regions {
		x1, x2 := maxInt(0, r[0]-1), minInt(w-1, r[1]+1)
		y1, y2 := maxInt(0, topY-1), minInt(h-1, botY+1)
		sub := make([][]int, y2-y1+1)
		for y := y1; y <= y2; y++ {
			row := make([]int, x2-x1+1)
			for x := x1; x <= x2; x++ {
				if bin[y][x] {
					row[x-x1] = gray[y][x]
				} else {
					row[x-x1] = 255
				}
			}
			sub[y-y1] = row
		}
		result = append(result, normalizeGray(sub))
	}
	return result
}

func findRegions(proj []int) [][2]int {
	regs := [][2]int{}
	s := -1
	for x, v := range proj {
		if v > 0 && s < 0 {
			s = x
		} else if v == 0 && s >= 0 {
			regs = append(regs, [2]int{s, x - 1})
			s = -1
		}
	}
	if s >= 0 {
		regs = append(regs, [2]int{s, len(proj) - 1})
	}
	return regs
}

func widestRegion(regions [][2]int) int {
	mi, mw := -1, 0
	for i, r := range regions {
		if rw := r[1] - r[0] + 1; rw > mw {
			mw = rw
			mi = i
		}
	}
	return mi
}

func hasWide(regions [][2]int, expectedW int) bool {
	for _, r := range regions {
		if r[1]-r[0]+1 > int(float64(expectedW)*1.6) {
			return true
		}
	}
	return false
}

func normalizeGray(img [][]int) []uint8 {
	h, w := len(img), len(img[0])
	top, bot, left, right := h, 0, w, 0
	found := false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if img[y][x] < 200 {
				found = true
				if y < top {
					top = y
				}
				if y > bot {
					bot = y
				}
				if x < left {
					left = x
				}
				if x > right {
					right = x
				}
			}
		}
	}
	if !found || top >= bot || left >= right {
		return filledGray()
	}
	cH, cW := bot-top+1, right-left+1
	trimmed := make([][]int, cH)
	for y := 0; y < cH; y++ {
		trimmed[y] = append([]int(nil), img[top+y][left:right+1]...)
	}
	pad, maxW, maxH := 1, ocrStdW-2, ocrStdH-2
	scale := math.Min(float64(maxW)/float64(cW), float64(maxH)/float64(cH))
	nW := maxInt(1, minInt(maxW, int(float64(cW)*scale)))
	nH := maxInt(1, minInt(maxH, int(float64(cH)*scale)))
	resized := make([][]int, nH)
	for y := 0; y < nH; y++ {
		resized[y] = make([]int, nW)
		for x := 0; x < nW; x++ {
			resized[y][x] = trimmed[y*cH/nH][x*cW/nW]
		}
	}
	result := make([]uint8, ocrStdW*ocrStdH)
	for i := range result {
		result[i] = 255
	}
	ox, oy := (ocrStdW-nW)/2, (ocrStdH-nH)/2
	_ = pad
	for y := 0; y < nH; y++ {
		for x := 0; x < nW; x++ {
			result[(y+oy)*ocrStdW+x+ox] = uint8(clampInt(resized[y][x], 0, 255))
		}
	}
	return result
}

func filledGray() []uint8 {
	out := make([]uint8, ocrStdW*ocrStdH)
	for i := range out {
		out[i] = 255
	}
	return out
}

func grayToBinary(gray []uint8) []bool {
	bin := make([]bool, len(gray))
	for i, v := range gray {
		bin[i] = v < 128
	}
	return bin
}

func classifyOCR(ch []uint8) byte {
	bin := grayToBinary(ch)
	feat := ocrFeatures(bin, ocrStdH, ocrStdW)
	bestNcc, bestIou, bestComb := -1.0, -1.0, -1.0
	cNcc, cIou, cComb := byte('?'), byte('?'), byte('?')
	for _, sample := range ocrSamples {
		n := nccOCR(ch, sample.gray)
		io := iouOCR(bin, sample.bin)
		fs := featSimOCR(feat, sample.feat)
		if n > bestNcc {
			bestNcc = n
			cNcc = sample.ch
		}
		if io > bestIou {
			bestIou = io
			cIou = sample.ch
		}
		comb := 0.25*n + 0.45*io + 0.30*fs
		if comb > bestComb {
			bestComb = comb
			cComb = sample.ch
		}
	}
	if cNcc == cIou || cNcc == cComb {
		return cNcc
	}
	return cComb
}

func ocrFeatures(bin []bool, h, w int) []float64 {
	total := 0
	rowD := make([]int, h)
	colD := make([]int, w)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if bin[y*w+x] {
				total++
				rowD[y]++
				colD[x]++
			}
		}
	}
	f := make([]float64, 20)
	if total == 0 {
		return f
	}
	f[0] = float64(total) / float64(h*w)
	topH, leftH := 0, 0
	for y := 0; y < h/2; y++ {
		topH += rowD[y]
	}
	for x := 0; x < w/2; x++ {
		leftH += colD[x]
	}
	f[1], f[2] = float64(topH)/float64(total), float64(leftH)/float64(total)
	for i := 0; i < 3; i++ {
		y := h * (i + 1) / 4
		cnt := 0
		prev := false
		for x := 0; x < w; x++ {
			if bin[y*w+x] != prev {
				cnt++
				prev = bin[y*w+x]
			}
		}
		f[3+i] = float64(cnt)
	}
	for i := 0; i < 3; i++ {
		x := w * (i + 1) / 4
		cnt := 0
		prev := false
		for y := 0; y < h; y++ {
			if bin[y*w+x] != prev {
				cnt++
				prev = bin[y*w+x]
			}
		}
		f[6+i] = float64(cnt)
	}
	q := [4]int{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if bin[y*w+x] {
				if y < h/2 && x < w/2 {
					q[0]++
				} else if y < h/2 {
					q[1]++
				} else if x < w/2 {
					q[2]++
				} else {
					q[3]++
				}
			}
		}
	}
	f[9], f[10], f[11] = float64(q[0])/float64(total), float64(q[1])/float64(total), float64(q[2])/float64(total)
	rowsFg, maxRow := 0, 0
	for y := 0; y < h; y++ {
		if rowD[y] > 0 {
			rowsFg++
		}
		if rowD[y] > maxRow {
			maxRow = rowD[y]
		}
	}
	if rowsFg > 0 {
		f[12] = float64(total) / float64(rowsFg)
	}
	f[13] = float64(maxRow)
	for i := 0; i < 5; i++ {
		band := 0
		for y := h * i / 5; y < h*(i+1)/5; y++ {
			band += rowD[y]
		}
		f[14+i] = float64(band) / float64(total)
	}
	f[19] = float64(w) / float64(h)
	return f
}

func featSimOCR(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return 1.0 / (1.0 + math.Sqrt(sum))
}

func nccOCR(a, b []uint8) float64 {
	n := len(a)
	sA, sB, sAA, sBB, sAB := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		va, vb := float64(a[i]), float64(b[i])
		sA += va
		sB += vb
		sAA += va * va
		sBB += vb * vb
		sAB += va * vb
	}
	mA, mB := sA/float64(n), sB/float64(n)
	vA, vB := sAA/float64(n)-mA*mA, sBB/float64(n)-mB*mB
	cov := sAB/float64(n) - mA*mB
	d := math.Sqrt(vA * vB)
	if d < 1e-6 {
		return 0
	}
	return (cov/d + 1) / 2
}

func iouOCR(a, b []bool) float64 {
	fA, fB, fAB := 0, 0, 0
	for i := range a {
		if a[i] {
			fA++
		}
		if b[i] {
			fB++
		}
		if a[i] && b[i] {
			fAB++
		}
	}
	denom := fA + fB - fAB
	if denom <= 0 {
		return 0
	}
	return float64(fAB) / float64(denom)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
