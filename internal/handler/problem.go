package handler

import (
	"encoding/json"
	"net/http"

	api "github.com/nestera-sh/template/internal/generated"
)

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	problemType, title, code, detail string,
) {
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + problemType,
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: stringPtr(r.URL.Path),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func stringPtr(s string) *string { return &s }

func respondBadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	respondProblem(w, r, http.StatusBadRequest,
		"invalid-request", "Invalid request", "invalid_request", detail)
}

func respondTripNotFound(w http.ResponseWriter, r *http.Request) {
	respondProblem(w, r, http.StatusNotFound,
		"trip-not-found", "Trip not found", "trip_not_found", "Trip was not found")
}

func respondTripCompleted(w http.ResponseWriter, r *http.Request) {
	respondProblem(w, r, http.StatusConflict,
		"trip-completed", "Trip completed", "trip_completed",
		"Operation is not allowed for a completed trip")
}

func respondDriverBusy(w http.ResponseWriter, r *http.Request) {
	respondProblem(w, r, http.StatusConflict,
		"driver-busy", "Driver busy", "driver_busy",
		"Driver already has an active trip")
}

func respondInternalError(w http.ResponseWriter, r *http.Request) {
	respondProblem(w, r, http.StatusInternalServerError,
		"internal-error", "Internal Server Error", "internal_error",
		"Internal server error")
}

func RespondInvalidParam(w http.ResponseWriter, r *http.Request, err error) {
	respondBadRequest(w, r, err.Error())
}
