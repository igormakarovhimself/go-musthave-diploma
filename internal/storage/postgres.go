package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

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

func (s *PostgresStorage) GetBalance(ctx context.Context, userID int64) (*models.Balance, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN status = 'PROCESSED' THEN accrual ELSE 0 END), 0) as total_accrual,
			COALESCE((SELECT SUM(sum) FROM withdrawals WHERE user_id = $1), 0) as total_withdrawn
		FROM orders
		WHERE user_id = $1
	`

	var result struct {
		TotalAccrual   sql.NullString `db:"total_accrual"`
		TotalWithdrawn sql.NullString `db:"total_withdrawn"`
	}

	err := s.db.GetContext(ctx, &result, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	totalAccrual, err := parseDecimal(result.TotalAccrual)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total accrual: %w", err)
	}

	totalWithdrawn, err := parseDecimal(result.TotalWithdrawn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total withdrawn: %w", err)
	}

	return &models.Balance{
		Current:   totalAccrual.Sub(totalWithdrawn),
		Withdrawn: totalWithdrawn,
	}, nil
}

func parseDecimal(ns sql.NullString) (decimal.Decimal, error) {
	if !ns.Valid || ns.String == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(ns.String)
}

func (s *PostgresStorage) calculateLockedBalance(ctx context.Context, tx *sqlx.Tx, userID int64) (decimal.Decimal, error) {
	_, err := tx.ExecContext(ctx,
		`SELECT number FROM orders 
		 WHERE user_id = $1 AND status = 'PROCESSED' FOR UPDATE`, userID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to lock orders: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`SELECT id FROM withdrawals 
		 WHERE user_id = $1 FOR UPDATE`, userID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to lock withdrawals: %w", err)
	}

	var accrualStr string
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(accrual), 0)::text FROM orders 
		 WHERE user_id = $1 AND status = 'PROCESSED'`, userID).Scan(&accrualStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to sum accruals: %w", err)
	}

	totalAccrual := decimal.Zero
	if accrualStr != "" && accrualStr != "0" {
		totalAccrual, err = decimal.NewFromString(accrualStr)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to parse accrual: %w", err)
		}
	}

	var withdrawnStr string
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(sum), 0)::text FROM withdrawals 
		 WHERE user_id = $1`, userID).Scan(&withdrawnStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to sum withdrawals: %w", err)
	}

	totalWithdrawn := decimal.Zero
	if withdrawnStr != "" && withdrawnStr != "0" {
		totalWithdrawn, err = decimal.NewFromString(withdrawnStr)
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to parse withdrawn: %w", err)
		}
	}

	return totalAccrual.Sub(totalWithdrawn), nil
}

func (s *PostgresStorage) CreateWithdrawal(ctx context.Context, userID int64, order string, sum decimal.Decimal) (*models.Withdrawal, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	currentBalance, err := s.calculateLockedBalance(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	if currentBalance.LessThan(sum) {
		return nil, ErrInsufficientFunds
	}

	query := `INSERT INTO withdrawals (user_id, "order", sum, processed_at)
			  VALUES ($1, $2, $3, NOW())
			  RETURNING id, user_id, "order", sum, processed_at`

	var withdrawal models.Withdrawal
	err = tx.QueryRowxContext(ctx, query, userID, order, sum).StructScan(&withdrawal)
	if err != nil {
		return nil, fmt.Errorf("failed to insert withdrawal: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &withdrawal, nil
}

func (s *PostgresStorage) GetWithdrawalsByUserID(ctx context.Context, userID int64) ([]*models.Withdrawal, error) {
	query := `SELECT id, user_id, "order", sum, processed_at 
			  FROM withdrawals 
			  WHERE user_id = $1 
			  ORDER BY processed_at DESC`

	var withdrawals []*models.Withdrawal
	err := s.db.SelectContext(ctx, &withdrawals, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}

	return withdrawals, nil
}

func (s *PostgresStorage) UpdateOrderStatus(ctx context.Context, number string, status string, accrual *decimal.Decimal) error {
	query := `UPDATE orders 
			  SET status = $2, accrual = $3
			  WHERE number = $1 AND status NOT IN ('INVALID', 'PROCESSED')
			  RETURNING number`

	var returnedNumber string
	err := s.db.QueryRowContext(ctx, query, number, status, accrual).Scan(&returnedNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetNextOrderForProcessing(ctx context.Context) (*models.Order, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `SELECT number, user_id, status, accrual, uploaded_at
			  FROM orders
			  WHERE status IN ('NEW', 'PROCESSING')
			  ORDER BY uploaded_at ASC
			  LIMIT 1
			  FOR UPDATE SKIP LOCKED`

	var order models.Order
	err = tx.GetContext(ctx, &order, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get next order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &order, nil
}
