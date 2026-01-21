package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go-musthave-diploma/internal/auth"
	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/luhn"
	"go-musthave-diploma/internal/middleware"
	"go-musthave-diploma/internal/storage"

	"github.com/shopspring/decimal"
)

type Handler struct {
	userRepo       storage.UserRepository
	orderRepo      storage.OrderRepository
	balanceRepo    storage.BalanceRepository
	withdrawalRepo storage.WithdrawalRepository
	jwtSecret      string
}

func NewHandler(userRepo storage.UserRepository, orderRepo storage.OrderRepository, balanceRepo storage.BalanceRepository, withdrawalRepo storage.WithdrawalRepository, jwtSecret string) *Handler {
	return &Handler{
		userRepo:       userRepo,
		orderRepo:      orderRepo,
		balanceRepo:    balanceRepo,
		withdrawalRepo: withdrawalRepo,
		jwtSecret:      jwtSecret,
	}
}

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "login and password are required", http.StatusBadRequest)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		logger.Log.Error("failed to hash password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.CreateUser(r.Context(), req.Login, passwordHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			http.Error(w, "user already exists", http.StatusConflict)
			return
		}
		logger.Log.Error("failed to create user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		logger.Log.Error("failed to generate token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	orderNumber := strings.TrimSpace(string(body))

	if orderNumber == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if !luhn.Validate(orderNumber) {
		http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orderNumber = strings.ReplaceAll(orderNumber, " ", "")

	_, err = h.orderRepo.CreateOrder(r.Context(), userID, orderNumber)
	if err != nil {
		if errors.Is(err, storage.ErrOrderExistsSameUser) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, storage.ErrOrderExistsOtherUser) {
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
			return
		}
		logger.Log.Error("failed to create order", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "login and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetUserByLogin(r.Context(), req.Login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		logger.Log.Error("failed to get user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		logger.Log.Error("failed to generate token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.orderRepo.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get orders", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.balanceRepo.GetBalance(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get balance", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if !luhn.Validate(req.Order) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	sum := decimal.NewFromFloat(req.Sum)
	_, err = h.withdrawalRepo.CreateWithdrawal(r.Context(), userID, req.Order, sum)
	if err != nil {
		if errors.Is(err, storage.ErrInsufficientFunds) {
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
			return
		}
		logger.Log.Error("failed to create withdrawal", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.withdrawalRepo.GetWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get withdrawals", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}
