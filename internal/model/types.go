package model

// CourseRequest represents the incoming API request
type CourseRequest struct {
	School   string `json:"school"`
	Account  string `json:"account"`
	Password string `json:"password"`
}

// CourseResponse represents the API response
type CourseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	MsgZhcn string      `json:"msg_zhcn,omitempty"`
	MsgEn  string      `json:"msg_en,omitempty"`
}
