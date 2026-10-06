package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/response"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (h *HealthController) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	writeJSON(w, http.StatusOK, response.HealthResponse{
		Status: "ok",
	})
}

func (h *HealthController) HandleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	writeJSON(w, http.StatusOK, response.InfoResponse{
		App:         "TLUAgent Web Service",
		Description: "TLUAgent Academic Advisory Service",
	})
}
