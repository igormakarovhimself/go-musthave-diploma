package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"go-musthave-diploma/internal/auth"
	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/middleware"
	"go-musthave-diploma/internal/models"
	"go-musthave-diploma/internal/storage"
)

func init() {
	logger.Initialize("error")
}

type mockUserRepo struct {
	users map[string]*models.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*models.User),
	}
}

func (m *mockUserRepo) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	if _, exists := m.users[login]; exists {
		return nil, storage.ErrUserExists
	}
	user := &models.User{
		ID:           int64(len(m.users) + 1),
		Login:        login,
		PasswordHash: passwordHash,
	}
	m.users[login] = user
	return user, nil
}

func (m *mockUserRepo) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	user, exists := m.users[login]
	if !exists {
		return nil, storage.ErrUserNotFound
	}
	return user, nil
}

type mockOrderRepo struct {
	orders map[string]*models.Order
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders: make(map[string]*models.Order),
	}
}

func (m *mockOrderRepo) CreateOrder(ctx context.Context, userID int64, orderNumber string) (*models.Order, error) {
	existing, ok := m.orders[orderNumber]
	if ok {
		if existing.UserID == userID {
			return existing, storage.ErrOrderExistsSameUser
		}
		return nil, storage.ErrOrderExistsOtherUser
	}

	order := &models.Order{
		Number:     orderNumber,
		UserID:     userID,
		Status:     models.OrderStatusNew,
		UploadedAt: time.Now(),
	}
	m.orders[orderNumber] = order
	return order, nil
}

func (m *mockOrderRepo) GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	order, exists := m.orders[orderNumber]
	if !exists {
		return nil, storage.ErrOrderNotFound
	}
	return order, nil
}

func (m *mockOrderRepo) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	var result []*models.Order
	for _, order := range m.orders {
		if order.UserID == userID {
			result = append(result, order)
		}
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].UploadedAt.Before(result[j].UploadedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

type mockBalanceRepo struct {
	balance *models.Balance
}

func newMockBalanceRepo() *mockBalanceRepo {
	return &mockBalanceRepo{
		balance: &models.Balance{
			Current:   decimal.Zero,
			Withdrawn: decimal.Zero,
		},
	}
}

func (m *mockBalanceRepo) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	return m.balance, nil
}

type errorUserRepo struct{}

func (e *errorUserRepo) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	return nil, errors.New("database error")
}

func (e *errorUserRepo) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	return nil, errors.New("database error")
}

type errorOrderRepo struct{}

func (e *errorOrderRepo) CreateOrder(ctx context.Context, userID int64, orderNumber string) (*models.Order, error) {
	return nil, errors.New("database error")
}

func (e *errorOrderRepo) GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	return nil, errors.New("database error")
}

func (e *errorOrderRepo) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	return nil, errors.New("database error")
}

type errorBalanceRepo struct{}

func (e *errorBalanceRepo) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	return nil, errors.New("database error")
}

func TestRegister(t *testing.T) {
	repo := newMockUserRepo()
	orderRepo := newMockOrderRepo()
	balanceRepo := newMockBalanceRepo()
	h := NewHandler(repo, orderRepo, balanceRepo, "test-secret")

	tests := []struct {
		name       string
		body       string
		wantStatus int
		setupRepo  func()
	}{
		{
			name:       "successful registration",
			body:       `{"login":"testuser","password":"testpass"}`,
			wantStatus: http.StatusOK,
			setupRepo:  func() {},
		},
		{
			name:       "duplicate login",
			body:       `{"login":"duplicate","password":"pass"}`,
			wantStatus: http.StatusConflict,
			setupRepo: func() {
				hash, _ := auth.HashPassword("pass")
				repo.CreateUser(context.Background(), "duplicate", hash)
			},
		},
		{
			name:       "invalid json",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
			setupRepo:  func() {},
		},
		{
			name:       "empty login",
			body:       `{"login":"","password":"pass"}`,
			wantStatus: http.StatusBadRequest,
			setupRepo:  func() {},
		},
		{
			name:       "empty password",
			body:       `{"login":"user","password":""}`,
			wantStatus: http.StatusBadRequest,
			setupRepo:  func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo.users = make(map[string]*models.User)
			tt.setupRepo()

			req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Register(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Register() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				authHeader := w.Header().Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Error("Expected Authorization header with Bearer token")
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	repo := newMockUserRepo()
	orderRepo := newMockOrderRepo()
	balanceRepo := newMockBalanceRepo()
	h := NewHandler(repo, orderRepo, balanceRepo, "test-secret")

	hash, _ := auth.HashPassword("correctpass")
	repo.CreateUser(context.Background(), "existinguser", hash)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "successful login",
			body:       `{"login":"existinguser","password":"correctpass"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong password",
			body:       `{"login":"existinguser","password":"wrongpass"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found",
			body:       `{"login":"nonexistent","password":"pass"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty login",
			body:       `{"login":"","password":"pass"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Login(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Login() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				authHeader := w.Header().Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Error("Expected Authorization header with Bearer token")
				}
			}
		})
	}
}

func TestRegister_DatabaseError(t *testing.T) {
	h := NewHandler(&errorUserRepo{}, &errorOrderRepo{}, &errorBalanceRepo{}, "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"test","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLogin_DatabaseError(t *testing.T) {
	h := NewHandler(&errorUserRepo{}, &errorOrderRepo{}, &errorBalanceRepo{}, "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"test","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestUploadOrder(t *testing.T) {
	userRepo := newMockUserRepo()
	orderRepo := newMockOrderRepo()
	balanceRepo := newMockBalanceRepo()
	h := NewHandler(userRepo, orderRepo, balanceRepo, "test-secret")

	tests := []struct {
		name       string
		body       string
		userID     int64
		wantStatus int
		setupRepo  func()
	}{
		{
			name:       "new order accepted",
			body:       "12345678903",
			userID:     1,
			wantStatus: http.StatusAccepted,
			setupRepo:  func() {},
		},
		{
			name:       "order already uploaded by same user",
			body:       "79927398713",
			userID:     1,
			wantStatus: http.StatusOK,
			setupRepo: func() {
				orderRepo.CreateOrder(context.Background(), 1, "79927398713")
			},
		},
		{
			name:       "order already uploaded by other user",
			body:       "4532015112830366",
			userID:     2,
			wantStatus: http.StatusConflict,
			setupRepo: func() {
				orderRepo.CreateOrder(context.Background(), 1, "4532015112830366")
			},
		},
		{
			name:       "invalid luhn checksum",
			body:       "12345678904",
			userID:     1,
			wantStatus: http.StatusUnprocessableEntity,
			setupRepo:  func() {},
		},
		{
			name:       "empty body",
			body:       "",
			userID:     1,
			wantStatus: http.StatusBadRequest,
			setupRepo:  func() {},
		},
		{
			name:       "order with spaces",
			body:       "1234 5678 903",
			userID:     1,
			wantStatus: http.StatusAccepted,
			setupRepo:  func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo.orders = make(map[string]*models.Order)
			tt.setupRepo()

			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "text/plain")
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			h.UploadOrder(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("UploadOrder() status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestGetOrders(t *testing.T) {
	userRepo := newMockUserRepo()
	orderRepo := newMockOrderRepo()
	balanceRepo := newMockBalanceRepo()
	h := NewHandler(userRepo, orderRepo, balanceRepo, "test-secret")

	tests := []struct {
		name          string
		userID        int64
		wantStatus    int
		wantOrdersNum int
		setupRepo     func()
	}{
		{
			name:          "no orders",
			userID:        1,
			wantStatus:    http.StatusNoContent,
			wantOrdersNum: 0,
			setupRepo:     func() {},
		},
		{
			name:          "multiple orders sorted by time",
			userID:        1,
			wantStatus:    http.StatusOK,
			wantOrdersNum: 3,
			setupRepo: func() {
				time.Sleep(1 * time.Millisecond)
				orderRepo.CreateOrder(context.Background(), 1, "12345678903")
				time.Sleep(1 * time.Millisecond)
				orderRepo.CreateOrder(context.Background(), 1, "79927398713")
				time.Sleep(1 * time.Millisecond)
				orderRepo.CreateOrder(context.Background(), 1, "4532015112830366")
			},
		},
		{
			name:          "only current user orders",
			userID:        2,
			wantStatus:    http.StatusOK,
			wantOrdersNum: 1,
			setupRepo: func() {
				orderRepo.CreateOrder(context.Background(), 1, "12345678903")
				orderRepo.CreateOrder(context.Background(), 2, "79927398713")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo.orders = make(map[string]*models.Order)
			tt.setupRepo()

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			h.GetOrders(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetOrders() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("GetOrders() Content-Type = %s, want application/json", contentType)
				}

				body := w.Body.String()
				if body == "" {
					t.Error("GetOrders() returned empty body for status 200")
				}
			}

			if tt.wantStatus == http.StatusNoContent {
				if w.Body.Len() != 0 {
					t.Errorf("GetOrders() body should be empty for 204, got: %s", w.Body.String())
				}
			}
		})
	}
}

func TestGetOrders_DatabaseError(t *testing.T) {
	h := NewHandler(&errorUserRepo{}, &errorOrderRepo{}, &errorBalanceRepo{}, "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetOrders(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestGetBalance(t *testing.T) {
	userRepo := newMockUserRepo()
	orderRepo := newMockOrderRepo()
	balanceRepo := newMockBalanceRepo()
	h := NewHandler(userRepo, orderRepo, balanceRepo, "test-secret")

	tests := []struct {
		name       string
		userID     int64
		balance    *models.Balance
		wantStatus int
	}{
		{
			name:   "zero balance",
			userID: 1,
			balance: &models.Balance{
				Current:   decimal.Zero,
				Withdrawn: decimal.Zero,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "balance with accrual",
			userID: 1,
			balance: &models.Balance{
				Current:   decimal.NewFromFloat(500.5),
				Withdrawn: decimal.Zero,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "balance with withdrawals",
			userID: 1,
			balance: &models.Balance{
				Current:   decimal.NewFromFloat(458.5),
				Withdrawn: decimal.NewFromFloat(42),
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balanceRepo.balance = tt.balance

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			h.GetBalance(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetBalance() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("GetBalance() Content-Type = %s, want application/json", contentType)
				}

				var response models.Balance
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				if !response.Current.Equal(tt.balance.Current) {
					t.Errorf("GetBalance() current = %s, want %s", response.Current, tt.balance.Current)
				}

				if !response.Withdrawn.Equal(tt.balance.Withdrawn) {
					t.Errorf("GetBalance() withdrawn = %s, want %s", response.Withdrawn, tt.balance.Withdrawn)
				}
			}
		})
	}
}

func TestGetBalance_DatabaseError(t *testing.T) {
	h := NewHandler(&errorUserRepo{}, &errorOrderRepo{}, &errorBalanceRepo{}, "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetBalance(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
