package services

import (
	"context"
	"database/sql"
	"errors"
	"html"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tanok/tanok-web-api/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type authClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Services) Login(ctx context.Context, email, password string) (string, *models.UserPublic, error) {
	if email == "" || password == "" {
		return "", nil, errors.New("email and password are required")
	}

	var user models.UserPublic
	var hashedPassword string
	err := s.DB.QueryRow(ctx,
		`SELECT id, email, COALESCE(name, ''), permissions, password FROM "User" WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Permissions, &hashedPassword)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.generateJWT(user.ID, user.Email)
	if err != nil {
		return "", nil, err
	}

	return token, &user, nil
}

func (s *Services) GetUserByID(ctx context.Context, userID string) (*models.UserPublic, error) {
	var user models.UserPublic
	err := s.DB.QueryRow(ctx,
		`SELECT id, email, COALESCE(name, ''), permissions FROM "User" WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Permissions)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Services) ValidateToken(tokenString string) (*authClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &authClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.Config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*authClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *Services) generateJWT(userID, email string) (string, error) {
	claims := &authClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.Config.JWTSecret))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ComparePassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *Services) ResolveDefaultAuthorID(ctx context.Context) (string, error) {
	if s.Config.DailyPostsAuthorID != "" {
		return s.Config.DailyPostsAuthorID, nil
	}

	rows, err := s.DB.Query(ctx,
		`SELECT id, permissions FROM "User" ORDER BY "createdAt" ASC LIMIT 20`,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type candidate struct {
		id          string
		permissions string
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.permissions); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}

	for _, c := range candidates {
		if containsDashboard(c.permissions) {
			return c.id, nil
		}
	}
	if len(candidates) > 0 {
		return candidates[0].id, nil
	}

	return "", errors.New("no users in the database. create an admin user before running the daily-posts cron")
}

func containsDashboard(permissions string) bool {
	return len(permissions) > 0 &&
		(permissions[0:1] == "{" || permissions[0:1] == "[") &&
		len(permissions) > 16
}

var _ = sql.NullString{}
var _ = html.EscapeString
