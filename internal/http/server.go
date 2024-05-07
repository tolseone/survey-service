package server

import (
	"net/http"

	"github.com/google/uuid"

	"survey/internal/app"
)

// TODO: rename interface, add new functions
type Service interface {
	GetAnswer() (string, error)
	GetScore(userID uuid.UUID) (int, error)
}

type Server struct {
	App    *app.Application
	Server *http.Server
}

func NewServer(app *app.Application) *Server {
	srv := &http.Server{
		Addr: ":4444",
		// TODO: handler
		Handler: nil,
	}

	return &Server{
		App:    app,
		Server: srv,
	}
}
