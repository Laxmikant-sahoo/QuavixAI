package auth

import (
	"quavixAI/internal/modules/user"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(u *user.User) error {
	query := `INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(query, u.ID, u.Email, u.PasswordHash, u.Name, u.Role, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *Repository) GetByEmail(email string) (*user.User, error) {
	var u user.User
	err := r.db.Get(&u, "SELECT * FROM users WHERE email=$1", email)
	return &u, err
}

func (r *Repository) GetByID(id string) (*user.User, error) {
	var u user.User
	err := r.db.Get(&u, "SELECT * FROM users WHERE id=$1", id)
	return &u, err
}
