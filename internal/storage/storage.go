package storage

import (
	"context"
	"errors"

	"go-musthave-diploma/internal/models"

	"github.com/shopspring/decimal"
)

var (
	ErrUserExists           = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrOrderExistsSameUser  = errors.New("order already uploaded by this user")
	ErrOrderExistsOtherUser = errors.New("order already uploaded by another user")
	ErrOrderNotFound        = errors.New("order not found")
	ErrInsufficientFunds    = errors.New("insufficient funds")
)

type UserRepository interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, orderNumber string) (*models.Order, error)
	GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error)
}

type BalanceRepository interface {
	GetBalance(ctx context.Context, userID int64) (*models.Balance, error)
}

type WithdrawalRepository interface {
	CreateWithdrawal(ctx context.Context, userID int64, order string, sum decimal.Decimal) (*models.Withdrawal, error)
	GetWithdrawalsByUserID(ctx context.Context, userID int64) ([]*models.Withdrawal, error)
}
