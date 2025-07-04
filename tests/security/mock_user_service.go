package security

import (
	"context"
	"errors"

	"connex/internal/api/user"
)

// MockUserService is a mock implementation of the user service for testing
type MockUserService struct {
	users map[string]*user.User
}

// NewMockUserService creates a new mock user service
func NewMockUserService() *MockUserService {
	return &MockUserService{
		users: make(map[string]*user.User),
	}
}

// Create mocks user creation
func (s *MockUserService) Create(ctx context.Context, u *user.User) (*user.User, error) {
	// Check if user already exists
	if _, exists := s.users[u.Email]; exists {
		return nil, errors.New("user already exists")
	}

	// Set ID and timestamps
	u.ID = int64(len(s.users) + 1)
	// In a real implementation, these would be set by the database
	// For testing, we'll just use the current time

	// Store user
	s.users[u.Email] = u
	return u, nil
}

// List mocks user listing
func (s *MockUserService) List(ctx context.Context) ([]*user.User, error) {
	users := make([]*user.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users, nil
}

// Get mocks user retrieval by ID
func (s *MockUserService) Get(ctx context.Context, id int64) (*user.User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

// Update mocks user update
func (s *MockUserService) Update(ctx context.Context, u *user.User) (*user.User, error) {
	if _, exists := s.users[u.Email]; !exists {
		return nil, errors.New("user not found")
	}
	s.users[u.Email] = u
	return u, nil
}

// Delete mocks user deletion
func (s *MockUserService) Delete(ctx context.Context, id int64) error {
	for email, u := range s.users {
		if u.ID == id {
			delete(s.users, email)
			return nil
		}
	}
	return errors.New("user not found")
}

// GetByEmail mocks user retrieval by email
func (s *MockUserService) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if u, exists := s.users[email]; exists {
		return u, nil
	}
	return nil, errors.New("user not found")
}
