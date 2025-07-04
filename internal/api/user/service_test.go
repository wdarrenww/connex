package user

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &UserService{}

	tests := []struct {
		name    string
		user    *User
		setup   func()
		wantErr bool
	}{
		{
			name: "successful create",
			user: &User{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			setup: func() {
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("John Doe", "john@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(1, time.Now(), time.Now()))
			},
			wantErr: false,
		},
		{
			name: "database error",
			user: &User{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			setup: func() {
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("John Doe", "john@example.com").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			result, err := service.Create(context.Background(), tt.user)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.user.Name, result.Name)
				assert.Equal(t, tt.user.Email, result.Email)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &UserService{}

	tests := []struct {
		name    string
		id      int64
		setup   func()
		want    *User
		wantErr bool
	}{
		{
			name: "successful get",
			id:   1,
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"}).
					AddRow(1, "John Doe", "john@example.com", "hash", time.Now(), time.Now())
				mock.ExpectQuery("SELECT.*FROM users WHERE id").
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: &User{
				ID:    1,
				Name:  "John Doe",
				Email: "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "user not found",
			id:   999,
			setup: func() {
				mock.ExpectQuery("SELECT.*FROM users WHERE id").
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			result, err := service.Get(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.want.ID, result.ID)
				assert.Equal(t, tt.want.Name, result.Name)
				assert.Equal(t, tt.want.Email, result.Email)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_GetByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &UserService{}

	tests := []struct {
		name    string
		email   string
		setup   func()
		want    *User
		wantErr bool
	}{
		{
			name:  "successful get by email",
			email: "john@example.com",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"}).
					AddRow(1, "John Doe", "john@example.com", "hash", time.Now(), time.Now())
				mock.ExpectQuery("SELECT.*FROM users WHERE email").
					WithArgs("john@example.com").
					WillReturnRows(rows)
			},
			want: &User{
				ID:    1,
				Name:  "John Doe",
				Email: "john@example.com",
			},
			wantErr: false,
		},
		{
			name:  "user not found by email",
			email: "nonexistent@example.com",
			setup: func() {
				mock.ExpectQuery("SELECT.*FROM users WHERE email").
					WithArgs("nonexistent@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			result, err := service.GetByEmail(context.Background(), tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.want.ID, result.ID)
				assert.Equal(t, tt.want.Name, result.Name)
				assert.Equal(t, tt.want.Email, result.Email)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &UserService{}

	tests := []struct {
		name    string
		setup   func()
		want    []*User
		wantErr bool
	}{
		{
			name: "successful list",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"}).
					AddRow(1, "John Doe", "john@example.com", "hash", time.Now(), time.Now()).
					AddRow(2, "Jane Doe", "jane@example.com", "hash", time.Now(), time.Now())
				mock.ExpectQuery("SELECT.*FROM users ORDER BY id").
					WillReturnRows(rows)
			},
			want: []*User{
				{ID: 1, Name: "John Doe", Email: "john@example.com"},
				{ID: 2, Name: "Jane Doe", Email: "jane@example.com"},
			},
			wantErr: false,
		},
		{
			name: "empty list",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"})
				mock.ExpectQuery("SELECT.*FROM users ORDER BY id").
					WillReturnRows(rows)
			},
			want:    []*User{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			result, err := service.List(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, len(tt.want))
				for i, user := range result {
					assert.Equal(t, tt.want[i].ID, user.ID)
					assert.Equal(t, tt.want[i].Name, user.Name)
					assert.Equal(t, tt.want[i].Email, user.Email)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &UserService{}

	tests := []struct {
		name    string
		user    *User
		setup   func()
		wantErr bool
	}{
		{
			name: "successful update",
			user: &User{
				ID:    1,
				Name:  "John Updated",
				Email: "john.updated@example.com",
			},
			setup: func() {
				mock.ExpectQuery("UPDATE users").
					WithArgs("John Updated", "john.updated@example.com", 1).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
						AddRow(time.Now(), time.Now()))
			},
			wantErr: false,
		},
		{
			name: "database error",
			user: &User{
				ID:    1,
				Name:  "John Updated",
				Email: "john.updated@example.com",
			},
			setup: func() {
				mock.ExpectQuery("UPDATE users").
					WithArgs("John Updated", "john.updated@example.com", 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			result, err := service.Update(context.Background(), tt.user)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.user.ID, result.ID)
				assert.Equal(t, tt.user.Name, result.Name)
				assert.Equal(t, tt.user.Email, result.Email)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &UserService{}

	tests := []struct {
		name    string
		id      int64
		setup   func()
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   1,
			setup: func() {
				mock.ExpectExec("DELETE FROM users WHERE id").
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			id:   1,
			setup: func() {
				mock.ExpectExec("DELETE FROM users WHERE id").
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			err := service.Delete(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
