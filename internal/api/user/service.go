package user

import (
	"context"
	"database/sql"
	"errors"

	"connex/internal/db"
)

type Service interface {
	Create(ctx context.Context, u *User) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Get(ctx context.Context, id int64) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	Delete(ctx context.Context, id int64) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type UserService struct{}

func NewService() *UserService {
	return &UserService{}
}

func (s *UserService) Create(ctx context.Context, u *User) (*User, error) {
	q := `INSERT INTO users (name, email, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id, created_at, updated_at`
	db := db.Get()
	err := db.QueryRowContext(ctx, q, u.Name, u.Email).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) List(ctx context.Context) ([]*User, error) {
	q := `SELECT id, name, email, created_at, updated_at FROM users ORDER BY id`
	db := db.Get()
	rows, err := db.QueryxContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := new(User)
		if err := rows.StructScan(u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *UserService) Get(ctx context.Context, id int64) (*User, error) {
	q := `SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`
	db := db.Get()
	u := new(User)
	err := db.QueryRowxContext(ctx, q, id).StructScan(u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) Update(ctx context.Context, u *User) (*User, error) {
	q := `UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE id = $3 RETURNING created_at, updated_at`
	db := db.Get()
	err := db.QueryRowContext(ctx, q, u.Name, u.Email, u.ID).Scan(&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) Delete(ctx context.Context, id int64) error {
	q := `DELETE FROM users WHERE id = $1`
	db := db.Get()
	_, err := db.ExecContext(ctx, q, id)
	return err
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*User, error) {
	q := `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
	db := db.Get()
	u := new(User)
	err := db.QueryRowxContext(ctx, q, email).StructScan(u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
