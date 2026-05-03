package schools

import (
	"net/http"
	"strings"
	"time"

	"xykcb_server/internal/config"
	"xykcb_server/internal/httpclient"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

var schoolClient = httpclient.NewClient("https://jw.hnit.edu.cn/njwhd", 10*time.Second)
var schoolRawClient = &http.Client{Timeout: 10 * time.Second}

type HnitA struct{}

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
