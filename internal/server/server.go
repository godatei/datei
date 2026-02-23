package server

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type server struct {
	db *pgxpool.Pool
}

func NewServer(db *pgxpool.Pool) *server {
	return &server{db: db}
}

// GetApiV1Ping implements [StrictServerInterface].
func (s *server) GetApiV1Ping(
	ctx context.Context,
	request GetApiV1PingRequestObject,
) (GetApiV1PingResponseObject, error) {
	return GetApiV1Ping200JSONResponse{Ping: "pong"}, nil
}

// PostApiV1Ping implements [StrictServerInterface].
func (s *server) PostApiV1Ping(
	ctx context.Context,
	request PostApiV1PingRequestObject,
) (PostApiV1PingResponseObject, error) {
	return PostApiV1Ping200JSONResponse{Ping: request.Body.Ping}, nil
}

var _ StrictServerInterface = (*server)(nil)
