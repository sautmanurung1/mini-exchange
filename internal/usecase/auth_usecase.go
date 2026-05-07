package usecase

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"mini-exchange/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	userRepo domain.UserRepository
	secret   []byte
}

func NewAuthUseCase(userRepo domain.UserRepository) *AuthUseCase {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "very-secret-key"
	}
	return &AuthUseCase{
		userRepo: userRepo,
		secret:   []byte(secret),
	}
}

func (u *AuthUseCase) Register(ctx context.Context, username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{
		ID:       uuid.New().String(),
		Username: username,
		Password: string(hashedPassword),
	}

	return u.userRepo.Create(ctx, user)
}

func (u *AuthUseCase) Login(ctx context.Context, username, password string) (string, error) {
	user, err := u.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString(u.secret)
}

func (u *AuthUseCase) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return u.secret, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	return claims["user_id"].(string), nil
}
