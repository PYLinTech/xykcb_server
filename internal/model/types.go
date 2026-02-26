package model

type CourseRequest struct {
	School   string `json:"school"`
	Account  string `json:"account"`
	Password string `json:"password"`
}

type CourseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	MsgZhcn string      `json:"msg_zhcn,omitempty"`
	MsgEn  string      `json:"msg_en,omitempty"`
}
