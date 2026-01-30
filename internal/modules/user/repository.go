package user

import (
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(id string) (*User, error) {
	var u User
	err := r.db.Get(&u, "SELECT * FROM users WHERE id=$1", id)
	return &u, err
}

func (r *Repository) UpdateUser(u *User) error {
	query := `UPDATE users SET name=$1, api_key=$2, updated_at=$3 WHERE id=$4`
	_, err := r.db.Exec(query, u.Name, u.APIKey, u.UpdatedAt, u.ID)
	return err
}

func (r *Repository) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id=$1`
	_, err := r.db.Exec(query, id)
	return err
}
