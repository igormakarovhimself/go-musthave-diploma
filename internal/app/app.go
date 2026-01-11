package app

import (
	"net/http"

	"go-musthave-diploma/internal/handlers"
	"go-musthave-diploma/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.GzipHandle)

	h := handlers.NewHandler()

	r.Post("/api/user/register", h.Register)
	r.Post("/api/user/login", h.Login)

	return r
}
