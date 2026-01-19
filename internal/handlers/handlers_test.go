package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-musthave-diploma/internal/auth"
	"go-musthave-diploma/internal/logger"
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

func TestRegister(t *testing.T) {
	repo := newMockUserRepo()
	h := NewHandler(repo, "test-secret")

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
	h := NewHandler(repo, "test-secret")

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

type errorUserRepo struct{}

func (e *errorUserRepo) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	return nil, errors.New("database error")
}

func (e *errorUserRepo) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	return nil, errors.New("database error")
}

func TestRegister_DatabaseError(t *testing.T) {
	h := NewHandler(&errorUserRepo{}, "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"test","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLogin_DatabaseError(t *testing.T) {
	h := NewHandler(&errorUserRepo{}, "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"test","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
