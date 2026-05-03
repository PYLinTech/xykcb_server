package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string {
	if e == nil {
		return "nil error"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var ErrorCodeMap = map[string]AppError{
	"001": {Code: "001", Message: "缺少必要参数", Status: http.StatusBadRequest},
	"002": {Code: "002", Message: "不支持的学校", Status: http.StatusNotFound},
	"003": {Code: "003", Message: "账户或密码错误", Status: http.StatusUnauthorized},
	"004": {Code: "004", Message: "服务器内部错误", Status: http.StatusInternalServerError},
	"005": {Code: "005", Message: "不支持的HTTP方法", Status: http.StatusMethodNotAllowed},
	"006": {Code: "006", Message: "获取数据失败", Status: http.StatusInternalServerError},
	"007": {Code: "007", Message: "请求超时", Status: http.StatusGatewayTimeout},
	"008": {Code: "008", Message: "Token已过期", Status: http.StatusUnauthorized},
	"009": {Code: "009", Message: "频率超限", Status: http.StatusTooManyRequests},
}

func GetError(code string) *AppError {
	if err, ok := ErrorCodeMap[code]; ok {
		return &AppError{Code: err.Code, Message: err.Message, Status: err.Status}
	}
	return &AppError{Code: code, Message: "未知错误", Status: http.StatusInternalServerError}
}

func Wrap(err error, code string) *AppError {
	if err == nil {
		return nil
	}
	appErr := GetError(code)
	appErr.Message = err.Error()
	return appErr
}
