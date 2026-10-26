package domainerror

import (
	"errors"
	"net/http"

	"github.com/nestera-sh/template/internal/domain/trip"
	api "github.com/nestera-sh/template/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	switch {
	case errors.Is(err, trip.ErrTripNotFound):
		return http.StatusNotFound, api.Problem{
			Type:   "https://tripgo.example/problems/trip-not-found",
			Title:  "Trip not found",
			Status: http.StatusNotFound,
			Code:   "trip_not_found",
			Detail: stringPtr("Trip wasn't found!"),
		}
	case errors.Is(err, trip.ErrTripCompleted):
		return http.StatusConflict, api.Problem{
			Type:   "https://tripgo.example/problems/trip-completed",
			Title:  "Trip completed",
			Status: http.StatusConflict,
			Code:   "trip_completed",
			Detail: stringPtr("Operation isn't allowed for a completed trip"),
		}
	case errors.Is(err, trip.ErrDriverBusy):
		return http.StatusConflict, api.Problem{
			Type:   "https://tripgo.example/problems/driver-busy",
			Title:  "Driver busy",
			Status: http.StatusConflict,
			Code:   "driver_busy",
			Detail: stringPtr("Driver already has an active trip"),
		}
	default:
		return http.StatusInternalServerError, api.Problem{
			Type:   "https://tripgo.example/problems/internal-error",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Code:   "internal_error",
			Detail: stringPtr("Internal server error"),
		}
	}
}

func stringPtr(s string) *string { return &s }
