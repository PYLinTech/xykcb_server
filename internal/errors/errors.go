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
	"001": {Code: "001", Message: "请求参数错误", Status: http.StatusBadRequest},
	"002": {Code: "002", Message: "不支持的学校", Status: http.StatusNotFound},
	"003": {Code: "003", Message: "账户或密码错误", Status: http.StatusUnauthorized},
	"004": {Code: "004", Message: "服务器内部错误", Status: http.StatusInternalServerError},
	"005": {Code: "005", Message: "频率超限", Status: http.StatusTooManyRequests},
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
