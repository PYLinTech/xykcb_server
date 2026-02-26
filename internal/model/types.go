package model

type CourseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	MsgZhcn string      `json:"msg_zhcn,omitempty"`
	MsgEn  string      `json:"msg_en,omitempty"`
}
