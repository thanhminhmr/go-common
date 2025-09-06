package http

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

type ServerResponse interface {
	Render(writer http.ResponseWriter) error
}

type ServerErrorResponse struct {
	Status int
	Cause  error
}

func (e ServerErrorResponse) Render(writer http.ResponseWriter) error {
	header := writer.Header()
	header.Add("Content-Type", "text/plain; charset=utf-8")
	header.Add("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(e.Status)
	_, err := writer.Write([]byte(e.Cause.Error()))
	return err
}

func (e ServerErrorResponse) Error() string {
	return e.Cause.Error()
}

func (e ServerErrorResponse) MarshalZerologObject(event *zerolog.Event) {
	event.AnErr("cause", e.Cause).Int("Status", e.Status)
}

type ServerJsonResponse struct {
	Status   int
	Response any
}

func (r ServerJsonResponse) Render(writer http.ResponseWriter) error {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(r.Status)
	return json.NewEncoder(writer).Encode(r.Response)
}
