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
)

type Handler struct {
	userRepo  storage.UserRepository
	orderRepo storage.OrderRepository
	jwtSecret string
}

func NewHandler(userRepo storage.UserRepository, orderRepo storage.OrderRepository, jwtSecret string) *Handler {
	return &Handler{
		userRepo:  userRepo,
		orderRepo: orderRepo,
		jwtSecret: jwtSecret,
	}
}

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
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
