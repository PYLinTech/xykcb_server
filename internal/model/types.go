package model

type CourseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	DescKey string      `json:"desc_key,omitempty"`
}
