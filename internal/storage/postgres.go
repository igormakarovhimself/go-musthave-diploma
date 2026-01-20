package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"go-musthave-diploma/internal/models"
)

type PostgresStorage struct {
	db *sqlx.DB
}

func NewPostgresStorage(db *sqlx.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	query := `INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, login, password_hash`

	var user models.User
	err := s.db.QueryRowxContext(ctx, query, login, passwordHash).StructScan(&user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (s *PostgresStorage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `SELECT id, login, password_hash FROM users WHERE login = $1`

	var user models.User
	err := s.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (s *PostgresStorage) GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	query := `SELECT number, user_id, status, accrual, uploaded_at FROM orders WHERE number = $1`

	var order models.Order
	err := s.db.GetContext(ctx, &order, query, orderNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (s *PostgresStorage) CreateOrder(ctx context.Context, userID int64, orderNumber string) (*models.Order, error) {
	existing, err := s.GetOrderByNumber(ctx, orderNumber)
	if err == nil {
		if existing.UserID == userID {
			return existing, ErrOrderExistsSameUser
		}
		return nil, ErrOrderExistsOtherUser
	}
	if !errors.Is(err, ErrOrderNotFound) {
		return nil, err
	}

	query := `INSERT INTO orders (number, user_id, status, uploaded_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING number, user_id, status, accrual, uploaded_at`

	var order models.Order
	err = s.db.QueryRowxContext(ctx, query, orderNumber, userID, models.OrderStatusNew).StructScan(&order)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return &order, nil
}

func (s *PostgresStorage) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	query := `SELECT number, user_id, status, accrual, uploaded_at 
		FROM orders 
		WHERE user_id = $1 
		ORDER BY uploaded_at DESC`

	var orders []*models.Order
	err := s.db.SelectContext(ctx, &orders, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return orders, nil
}
