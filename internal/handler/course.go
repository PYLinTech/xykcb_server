package handler

import (
	"encoding/json"
	"net/http"
	"xykcb_server/internal/config"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type CourseHandler struct{ registry *provider.Registry }

func NewCourseHandler() *CourseHandler { return &CourseHandler{registry: provider.Default()} }

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	cors := config.GetCORSConfig()
	if cors.AllowAll {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		origin := r.Header.Get("Origin")
		for _, host := range cors.AllowedHosts {
			if host == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (h *CourseHandler) HandleCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions { setCORSHeaders(w, r); return }
	if r.Method != http.MethodGet {
		h.error(w, r, http.StatusMethodNotAllowed, "Method not allowed", "Method not allowed")
		return
	}

	school, account, password := r.URL.Query().Get("school"), r.URL.Query().Get("account"), r.URL.Query().Get("password")
	if school == "" || account == "" || password == "" {
		h.error(w, r, http.StatusBadRequest, "缺少必要参数：school, account, password", "Missing required parameters")
		return
	}

	p, ok := h.registry.Get(school)
	if !ok {
		h.error(w, r, http.StatusNotFound, "不支持的学校: "+school, "School not supported: "+school)
		return
	}

	resp, err := p.Login(account, password)
	if err != nil {
		h.error(w, r, http.StatusInternalServerError, err.Error(), err.Error())
		return
	}

	h.json(w, r, http.StatusOK, resp)
}

func (h *CourseHandler) GetSupportedSchools(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions { setCORSHeaders(w, r); return }
	if r.Method != http.MethodGet {
		h.error(w, r, http.StatusMethodNotAllowed, "Method not allowed", "Method not allowed")
		return
	}
	h.json(w, r, http.StatusOK, map[string]interface{}{"success": true, "data": h.registry.ListAll()})
}

func (h *CourseHandler) json(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	setCORSHeaders(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(data)
}

func (h *CourseHandler) error(w http.ResponseWriter, r *http.Request, code int, msgZhcn, msgEn string) {
	setCORSHeaders(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(model.CourseResponse{Success: false, MsgZhcn: msgZhcn, MsgEn: msgEn})
}
