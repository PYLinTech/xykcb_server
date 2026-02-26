package schools

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type HnitA struct{ token string }

func init() { provider.Default().Register(&HnitA{}) }

func (s *HnitA) GetSchoolId() string   { return "1" }
func (s *HnitA) GetNameZhcn() string   { return "湖南工学院（移动端）" }
func (s *HnitA) GetNameEn() string     { return "Hunan Institute Of Technology (Mobile)" }

func (s *HnitA) result(code int) *model.CourseResponse {
	switch code {
	case 0: return &model.CourseResponse{Success: true, Data: s.token}
	case 1: return &model.CourseResponse{Success: false, MsgZhcn: "账号或密码错误", MsgEn: "Invalid account or password"}
	default: return &model.CourseResponse{Success: false, MsgZhcn: "登录失败", MsgEn: "Login failed"}
	}
}

func (s *HnitA) encryptPassword(password string) string {
	key, _ := hex.DecodeString("717a6b6a316b6a6768643d383736262a")
	block, _ := aes.NewCipher(key)
	plain, size := []byte("\""+password+"\""), block.BlockSize()
	padding := size - len(plain)%size
	plain = append(plain, make([]byte, padding)...)
	for i := len(plain) - padding; i < len(plain); i++ { plain[i] = byte(padding) }
	encrypted := make([]byte, len(plain))
	for i := 0; i < len(plain); i += size { block.Encrypt(encrypted[i:i+size], plain[i:i+size]) }
	first := base64.StdEncoding.EncodeToString(encrypted)
	return base64.StdEncoding.EncodeToString([]byte(first))
}

func (s *HnitA) Login(account, password string) (*model.CourseResponse, error) {
	resp, err := http.Post("https://jw.hnit.edu.cn/njwhd/login?userNo="+account+"&pwd="+s.encryptPassword(password), "", nil)
	if err != nil { return s.result(2), nil }
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return s.result(2), nil }

	code := data["code"].(string)
	if code != "1" { if strings.Contains(data["Msg"].(string), "密码错误") { return s.result(1), nil }; return s.result(2), nil }
	s.token = data["data"].(map[string]interface{})["token"].(string)
	return s.result(0), nil
}
