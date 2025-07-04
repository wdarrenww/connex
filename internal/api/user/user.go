package user

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type User struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	PasswordHash string    `db:"password_hash" json:"-"`
}

// Validate basic user fields for create/update
func (u *User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(u.Name) > 100 {
		return fmt.Errorf("name too long")
	}
	if !isValidEmail(u.Email) {
		return fmt.Errorf("invalid email format")
	}
	if len(u.Email) > 255 {
		return fmt.Errorf("email too long")
	}
	if strings.ContainsAny(u.Name, "<>\"'&") {
		return fmt.Errorf("name contains invalid characters")
	}
	return nil
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
