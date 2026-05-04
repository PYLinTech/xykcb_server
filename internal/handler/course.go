package handler

import (
	"encoding/json"
	"net/http"
	"xykcb_server/internal/config"
	"xykcb_server/internal/errors"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type schoolProviderCall func(provider.SchoolProvider, string, string) (*model.CourseResponse, error)

type CourseHandler struct {
	registry *provider.Registry
}

func NewCourseHandler() *CourseHandler {
	return &CourseHandler{registry: provider.Default()}
}

func (h *CourseHandler) json(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(data)
}

func (h *CourseHandler) error(w http.ResponseWriter, err *errors.AppError) {
	writeError(w, err)
}

func (h *CourseHandler) requireGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return false
	}
	if r.Method != http.MethodGet {
		h.error(w, errors.GetError("001"))
		return false
	}
	return true
}

func readCredentials(r *http.Request) (school, account, password string) {
	q := r.URL.Query()
	return q.Get("school"), q.Get("account"), q.Get("password")
}

func (h *CourseHandler) callSchoolProvider(w http.ResponseWriter, school, account, password string, call schoolProviderCall) (*model.CourseResponse, bool) {
	p, ok := h.registry.Get(school)
	if !ok {
		h.error(w, errors.GetError("002"))
		return nil, false
	}

	resp, err := call(p, account, password)
	if err != nil {
		h.error(w, errors.Wrap(err, "004"))
		return nil, false
	}

	if resp == nil || !resp.Success {
		descKey := "004"
		if resp != nil {
			descKey = resp.DescKey
		}
		h.error(w, errors.GetError(descKey))
		return nil, false
	}

	return resp, true
}

func (h *CourseHandler) HandleCourse(w http.ResponseWriter, r *http.Request) {
	if !h.requireGet(w, r) {
		return
	}

	school, account, password := readCredentials(r)
	if school == "" || account == "" || password == "" {
		h.error(w, errors.GetError("001"))
		return
	}

	resp, ok := h.callSchoolProvider(w, school, account, password, func(p provider.SchoolProvider, account, password string) (*model.CourseResponse, error) {
		return p.Login(account, password)
	})
	if !ok {
		return
	}

	data, ok := resp.Data.(string)
	if !ok {
		h.error(w, errors.GetError("004"))
		return
	}

	h.json(w, http.StatusOK, &model.CourseResponse{Success: true, Data: data})
}

func (h *CourseHandler) HandleCourseGrades(w http.ResponseWriter, r *http.Request) {
	if !h.requireGet(w, r) {
		return
	}

	school, account, password := readCredentials(r)
	semester := r.URL.Query().Get("semester")
	if school == "" || account == "" || password == "" {
		h.error(w, errors.GetError("001"))
		return
	}

	resp, ok := h.callSchoolProvider(w, school, account, password, func(p provider.SchoolProvider, account, password string) (*model.CourseResponse, error) {
		return p.GetGrades(account, password, semester)
	})
	if !ok {
		return
	}

	h.json(w, http.StatusOK, resp)
}

func (h *CourseHandler) HandleGuidanceTeaching(w http.ResponseWriter, r *http.Request) {
	if !h.requireGet(w, r) {
		return
	}

	school, account, password := readCredentials(r)
	if school == "" || account == "" || password == "" {
		h.error(w, errors.GetError("001"))
		return
	}

	resp, ok := h.callSchoolProvider(w, school, account, password, func(p provider.SchoolProvider, account, password string) (*model.CourseResponse, error) {
		return p.GetGuidanceTeaching(account, password)
	})
	if !ok {
		return
	}

	h.json(w, http.StatusOK, resp)
}

func (h *CourseHandler) GetSupportedSchools(w http.ResponseWriter, r *http.Request) {
	if !h.requireGet(w, r) {
		return
	}
	h.json(w, http.StatusOK, map[string]interface{}{"success": true, "data": h.registry.ListAll()})
}

func (h *CourseHandler) GetSupportFunctions(w http.ResponseWriter, r *http.Request) {
	if !h.requireGet(w, r) {
		return
	}

	school := r.URL.Query().Get("school")
	if school == "" {
		h.error(w, errors.GetError("001"))
		return
	}

	if config.GetSchoolConfigById(school) == nil {
		h.error(w, errors.GetError("002"))
		return
	}

	functions := config.GetSchoolFunctionsById(school)
	h.json(w, http.StatusOK, map[string]interface{}{"success": true, "data": functions})
}
