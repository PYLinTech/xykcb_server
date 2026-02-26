package handler

import (
	"encoding/json"
	"net/http"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

// CourseHandler handles course-related requests
type CourseHandler struct {
	registry *provider.Registry
}

// NewCourseHandler creates a new course handler
func NewCourseHandler() *CourseHandler {
	return &CourseHandler{
		registry: provider.Default(),
	}
}

// setCORSHeaders sets CORS headers for cross-origin requests
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// HandleCourse processes the /api/auto request
func (h *CourseHandler) HandleCourse(w http.ResponseWriter, r *http.Request) {
	// Handle preflight request
	if r.Method == http.MethodOptions {
		setCORSHeaders(w)
		return
	}

	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	school := r.URL.Query().Get("school")
	account := r.URL.Query().Get("account")
	password := r.URL.Query().Get("password")

	// Validate required parameters
	if school == "" || account == "" || password == "" {
		resp := model.CourseResponse{
			Success: false,
			MsgZhcn: "缺少必要参数：school, account, password",
			MsgEn:   "Missing required parameters: school, account, password",
		}
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Get the school provider
	p, ok := h.registry.Get(school)
	if !ok {
		resp := model.CourseResponse{
			Success: false,
			MsgZhcn: "不支持的学校: " + school,
			MsgEn:   "School not supported: " + school,
		}
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Call the school's login method
	resp, err := p.Login(account, password)
	if err != nil {
		resp := model.CourseResponse{
			Success: false,
			MsgZhcn: err.Error(),
			MsgEn:   err.Error(),
		}
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Return the response
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetSupportedSchools returns the list of supported schools
func (h *CourseHandler) GetSupportedSchools(w http.ResponseWriter, r *http.Request) {
	// Handle preflight request
	if r.Method == http.MethodOptions {
		setCORSHeaders(w)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	schools := h.registry.ListAll()
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    schools,
	})
}
