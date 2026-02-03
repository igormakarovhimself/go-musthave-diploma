package app

import (
	"net/http"

	"go-musthave-diploma/internal/handlers"
	"go-musthave-diploma/internal/middleware"
	"go-musthave-diploma/internal/storage"

	"github.com/go-chi/chi/v5"
)

func NewRouter(userRepo storage.UserRepository, orderRepo storage.OrderRepository, balanceRepo storage.BalanceRepository, withdrawalRepo storage.WithdrawalRepository, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.GzipHandle)

	h := handlers.NewHandler(userRepo, orderRepo, balanceRepo, withdrawalRepo, jwtSecret)

	r.Post("/api/user/register", h.Register)
	r.Post("/api/user/login", h.Login)

	r.With(middleware.Auth(jwtSecret)).Post("/api/user/orders", h.UploadOrder)
	r.With(middleware.Auth(jwtSecret)).Get("/api/user/orders", h.GetOrders)
	r.With(middleware.Auth(jwtSecret)).Get("/api/user/balance", h.GetBalance)
	r.With(middleware.Auth(jwtSecret)).Post("/api/user/balance/withdraw", h.Withdraw)
	r.With(middleware.Auth(jwtSecret)).Get("/api/user/withdrawals", h.GetWithdrawals)

	return r
}
