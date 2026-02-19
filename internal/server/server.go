package server

import (
	"encoding/json"
	"net/http"

	"github.com/godatei/datei/pkg/api"
)

type server struct{}

func NewServer() *server {
	return &server{}
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
