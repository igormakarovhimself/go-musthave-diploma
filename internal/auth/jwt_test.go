package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestGenerateToken(t *testing.T) {
	secret := "test-secret"
	userID := int64(123)

	token, err := GenerateToken(userID, secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Error("Token is empty")
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	userID := int64(123)

	token, err := GenerateToken(userID, secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected userID %d, got %d", userID, claims.UserID)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	wrongSecret := "wrong-secret"
	userID := int64(123)

	token, err := GenerateToken(userID, secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ValidateToken(token, wrongSecret)
	if err == nil {
		t.Error("Expected error for invalid signature, got nil")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	userID := int64(123)

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	_, err = ValidateToken(tokenString, secret)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	secret := "test-secret"
	userID := int64(123)

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	_, err = ValidateToken(tokenString, secret)
	if err == nil {
		t.Error("Expected error for wrong signing method, got nil")
	}
}
