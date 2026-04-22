package handler

import (
	"encoding/json"
	"net/http"
	"xykcb_server/internal/config"
	"xykcb_server/internal/errors"
	"xykcb_server/internal/model"
	"xykcb_server/internal/provider"
)

type CourseHandler struct {
	registry *provider.Registry
}

func NewCourseHandler() *CourseHandler {
	return &CourseHandler{registry: provider.Default()}
}

func (h *CourseHandler) json(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(data)
}

func (h *CourseHandler) error(w http.ResponseWriter, r *http.Request, err *errors.AppError) {
	writeError(w, r, err)
}

func (h *CourseHandler) HandleCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		h.error(w, r, errors.GetError("005"))
		return
	}

	school, account, password := r.URL.Query().Get("school"), r.URL.Query().Get("account"), r.URL.Query().Get("password")
	if school == "" || account == "" || password == "" {
		h.error(w, r, errors.GetError("001"))
		return
	}

	h.handleSchoolRequest(w, r, school, account, password, nil, func(p provider.SchoolProvider, account, password string, semester string) (*model.CourseResponse, error) {
		return p.Login(account, password)
	})
}

func (h *CourseHandler) HandleCourseGrades(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		h.error(w, r, errors.GetError("005"))
		return
	}

	school, account, password, semester := r.URL.Query().Get("school"), r.URL.Query().Get("account"), r.URL.Query().Get("password"), r.URL.Query().Get("semester")
	if school == "" || account == "" || password == "" {
		h.error(w, r, errors.GetError("001"))
		return
	}

	h.handleSchoolRequest(w, r, school, account, password, &semester, func(p provider.SchoolProvider, account, password string, semester string) (*model.CourseResponse, error) {
		return p.GetGrades(account, password, semester)
	})
}

func (h *CourseHandler) HandleGuidanceTeaching(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		h.error(w, r, errors.GetError("005"))
		return
	}

	school, account, password := r.URL.Query().Get("school"), r.URL.Query().Get("account"), r.URL.Query().Get("password")
	if school == "" || account == "" || password == "" {
		h.error(w, r, errors.GetError("001"))
		return
	}

	h.handleSchoolRequest(w, r, school, account, password, nil, func(p provider.SchoolProvider, account, password string, semester string) (*model.CourseResponse, error) {
		return p.GetGuidanceTeaching(account, password)
	})
}

func (h *CourseHandler) handleSchoolRequest(w http.ResponseWriter, r *http.Request, school, account, password string, semester *string, handler func(provider.SchoolProvider, string, string, string) (*model.CourseResponse, error)) {
	p, ok := h.registry.Get(school)
	if !ok {
		h.error(w, r, errors.GetError("002"))
		return
	}

	var resp *model.CourseResponse
	var err error

	if semester != nil && *semester != "" {
		resp, err = handler(p, account, password, *semester)
	} else {
		resp, err = handler(p, account, password, "")
	}

	if err != nil {
		h.error(w, r, errors.Wrap(err, "006"))
		return
	}

	if resp == nil || !resp.Success {
		descKey := "004"
		if resp != nil {
			descKey = resp.DescKey
		}
		h.error(w, r, errors.GetError(descKey))
		return
	}

	h.json(w, r, http.StatusOK, resp)
}

func (h *CourseHandler) GetSupportedSchools(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		h.error(w, r, errors.GetError("005"))
		return
	}
	h.json(w, r, http.StatusOK, map[string]interface{}{"success": true, "data": h.registry.ListAll()})
}

func (h *CourseHandler) GetSupportFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		h.error(w, r, errors.GetError("005"))
		return
	}

	school := r.URL.Query().Get("school")
	if school == "" {
		h.error(w, r, errors.GetError("001"))
		return
	}

	if config.GetSchoolConfigById(school) == nil {
		h.error(w, r, errors.GetError("002"))
		return
	}

	functions := config.GetSchoolFunctionsById(school)
	h.json(w, r, http.StatusOK, map[string]interface{}{"success": true, "data": functions})
}
