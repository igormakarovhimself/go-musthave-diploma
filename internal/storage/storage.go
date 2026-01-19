package storage

import (
	"context"
	"errors"

	"go-musthave-diploma/internal/models"
)

var (
	ErrUserExists           = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrOrderExistsSameUser  = errors.New("order already uploaded by this user")
	ErrOrderExistsOtherUser = errors.New("order already uploaded by another user")
	ErrOrderNotFound        = errors.New("order not found")
)

type UserRepository interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, orderNumber string) (*models.Order, error)
	GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error)
}
