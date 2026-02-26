package handler

import (
	"encoding/json"
	"net/http"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type CourseHandler struct{ registry *provider.Registry }

func NewCourseHandler() *CourseHandler { return &CourseHandler{registry: provider.Default()} }

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (h *CourseHandler) HandleCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions { setCORSHeaders(w); return }
	if r.Method != http.MethodGet { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }

	school, account, password := r.URL.Query().Get("school"), r.URL.Query().Get("account"), r.URL.Query().Get("password")
	if school == "" || account == "" || password == "" {
		resp := model.CourseResponse{Success: false, MsgZhcn: "缺少必要参数：school, account, password", MsgEn: "Missing required parameters"}
		setCORSHeaders(w); w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusBadRequest); json.NewEncoder(w).Encode(resp); return
	}

	p, ok := h.registry.Get(school)
	if !ok {
		resp := model.CourseResponse{Success: false, MsgZhcn: "不支持的学校: " + school, MsgEn: "School not supported: " + school}
		setCORSHeaders(w); w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusNotFound); json.NewEncoder(w).Encode(resp); return
	}

	resp, err := p.Login(account, password)
	if err != nil {
		resp := model.CourseResponse{Success: false, MsgZhcn: err.Error(), MsgEn: err.Error()}
		setCORSHeaders(w); w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusInternalServerError); json.NewEncoder(w).Encode(resp); return
	}

	setCORSHeaders(w); w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK); json.NewEncoder(w).Encode(resp)
}

func (h *CourseHandler) GetSupportedSchools(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions { setCORSHeaders(w); return }
	if r.Method != http.MethodGet { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	setCORSHeaders(w); w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK); json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": h.registry.ListAll()})
}
