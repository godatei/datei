package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/godatei/datei/pkg/api"
)

type server struct{}

func NewServer() *server1 {
	return &server1{}
}

func (s *server) PostApiV1Ping(w http.ResponseWriter, r *http.Request) {
	var ping api.Ping
	if err := json.NewDecoder(r.Body).Decode(&ping); err != nil {
		http.Error(w, "invalid ping", http.StatusBadRequest)
		return
	}

	resp := api.Pong(ping)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *server) GetApiV1Ping(w http.ResponseWriter, r *http.Request) {
	resp := api.Pong{Ping: "pong"}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

var _ ServerInterface = (*server)(nil)

type server1 struct{}

// GetApiV1Ping implements [StrictServerInterface].
func (s *server1) GetApiV1Ping(
	ctx context.Context,
	request GetApiV1PingRequestObject,
) (GetApiV1PingResponseObject, error) {
	return GetApiV1Ping200JSONResponse{Ping: "pong"}, nil
}

// PostApiV1Ping implements [StrictServerInterface].
func (s *server1) PostApiV1Ping(
	ctx context.Context,
	request PostApiV1PingRequestObject,
) (PostApiV1PingResponseObject, error) {
	return PostApiV1Ping200JSONResponse{Ping: request.Body.Ping}, nil
}

var _ StrictServerInterface = (*server1)(nil)
