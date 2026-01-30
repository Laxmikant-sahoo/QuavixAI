package auth

import (
	"errors"
	"time"

	"quavixAI/internal/modules/user"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
	jwt  JWTService
}

func NewService(r *Repository, jwt JWTService) *Service {
	return &Service{repo: r, jwt: jwt}
}

func (s *Service) Signup(email, password, name string) (*user.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u := &user.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         "user", // Default role
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(u); err != nil {
		return nil, "", err
	}

	token, err := s.jwt.Generate(u.ID, u.Role)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}

func (s *Service) Login(email, password string) (*user.User, string, error) {
	u, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, "", errors.New("invalid credentials")
	}

	token, err := s.jwt.Generate(u.ID, u.Role)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}
