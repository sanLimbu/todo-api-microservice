package rest

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/sanLimbu/todo-api/internal"
	"go.opentelemetry.io/otel"
)

const otelName = "github.com/sanLimbu/todo-api/internal/rest"

//ErrorResponse represents a response containing an error message
type ErrorResponse struct {
	Error       string            `json:"error"`
	Validations validation.Errors `json:"validations,omitempty"`
}

func renderErrorResponse(w http.ResponseWriter, r *http.Request, msg string, err error) {
	resp := ErrorResponse{Error: msg}
	status := http.StatusInternalServerError

	var ierr *internal.Error
	if !errors.As(err, &ierr) {
		resp.Error = "internal error"
	} else {
		switch ierr.Code() {
		case internal.ErrorCodeNotFound:
			status = http.StatusNotFound
		case internal.ErrorCodeInvalidArgument:
			status = http.StatusBadRequest

			var verrors validation.Errors
			if errors.As(ierr, &verrors) {
				resp.Validations = verrors
			}
		case internal.ErrorCodeUnkown:
			fallthrough
		default:
			status = http.StatusInternalServerError
		}
	}
	if err != nil {

		_, span := otel.Tracer(otelName).Start(r.Context(), "renderErrorResponse")
		defer span.End()

		span.RecordError(err)
	}
	render.Status(r, status)
	render.JSON(w, r, &resp)

}

func renderResponse(w http.ResponseWriter, r *http.Request, res interface{}, status int) {
	render.Status(r, status)
	render.JSON(w, r, res)
}
