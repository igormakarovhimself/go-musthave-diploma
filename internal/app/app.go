package app

import (
	"net/http"

	"go-musthave-diploma/internal/handlers"
	"go-musthave-diploma/internal/middleware"
	"go-musthave-diploma/internal/storage"

	"github.com/go-chi/chi/v5"
)

func NewRouter(userRepo storage.UserRepository, orderRepo storage.OrderRepository, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.GzipHandle)

	h := handlers.NewHandler(userRepo, orderRepo, jwtSecret)

	r.Post("/api/user/register", h.Register)
	r.Post("/api/user/login", h.Login)

	r.With(middleware.Auth(jwtSecret)).Post("/api/user/orders", h.UploadOrder)
	r.With(middleware.Auth(jwtSecret)).Get("/api/user/orders", h.GetOrders)

	return r
}
