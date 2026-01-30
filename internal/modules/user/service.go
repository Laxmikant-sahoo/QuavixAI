package user

import (
	"errors"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(r *Repository) *Service {
	return &Service{repo: r}
}

func (s *Service) GetUserProfile(userID string) (*User, error) {
	u, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (s *Service) UpdateUserProfile(userID string, req *ProfileUpdateRequest) (*User, error) {
	u, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	u.Name = req.Name
	u.APIKey = req.APIKey
	u.UpdatedAt = time.Now()

	if err := s.repo.UpdateUser(u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) DeleteUserProfile(userID string) error {
	err := s.repo.DeleteUser(userID)
	if err != nil {
		return errors.New("failed to delete user")
	}
	return nil
}
