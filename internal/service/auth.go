package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// JWTConfig holds configuration for JWT token generation
type JWTConfig struct {
	Secret    string
	ExpiresIn int // hours
}

// AuthService handles authentication and authorization
type AuthService struct {
	repos     *repository.Factory
	jwtConfig JWTConfig
}

// NewAuthService creates a new authentication service
func NewAuthService(repos *repository.Factory, jwtConfig JWTConfig) *AuthService {
	return &AuthService{
		repos:     repos,
		jwtConfig: jwtConfig,
	}
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *models.User, error) {
	// Get user by username
	user, err := s.repos.User.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return "", nil, fmt.Errorf("user account is inactive")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

// generateToken generates a JWT token for a user
func (s *AuthService) generateToken(userID uuid.UUID, role models.UserRole) (string, error) {
	expirationTime := time.Now().Add(time.Duration(s.jwtConfig.ExpiresIn) * time.Hour)

	claims := &Claims{
		UserID: userID.String(),
		Role:   string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetUserFromToken gets the user associated with a token
func (s *AuthService) GetUserFromToken(ctx context.Context, tokenString string) (*models.User, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// RegisterUser registers a new user
func (s *AuthService) RegisterUser(ctx context.Context, req models.UserRequest) (*models.User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user model
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         req.Role,
		IsActive:     req.IsActive,
	}

	// Create the user
	createdUser, err := s.repos.User.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return createdUser, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// Get the user
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword))
	if err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update the password
	err = s.repos.User.UpdatePassword(ctx, userID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
