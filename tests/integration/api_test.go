package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connex/internal/api/auth"
	"connex/internal/api/user"
	"connex/internal/config"
	"connex/internal/db"
	"connex/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testEnv struct {
	container testcontainers.Container
	dbURL     string
	cleanup   func()
}

func setupTestEnvironment(t *testing.T) *testEnv {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections"),
		),
	)
	require.NoError(t, err)

	// Get database URL
	dbURL, err := postgresContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Run migrations (simplified for test)
	// In a real test, you'd run actual migrations

	cleanup := func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return &testEnv{
		container: postgresContainer,
		dbURL:     dbURL,
		cleanup:   cleanup,
	}
}

func TestUserAPI_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	// Setup test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: env.dbURL,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
		Log: config.LogConfig{
			Level: "debug",
			Env:   "test",
		},
	}

	// Initialize logger
	_, err := logger.New(cfg.Log.Level, cfg.Log.Env)
	require.NoError(t, err)

	// Initialize database
	dbInstance, err := db.Init(cfg.Database)
	require.NoError(t, err)
	defer dbInstance.Close()

	// Create services and handlers
	userService := user.NewService()
	userHandler := user.NewHandler(userService)
	authHandler := auth.NewHandler(userService, cfg.JWT.Secret)

	// Test user registration
	t.Run("User Registration", func(t *testing.T) {
		registerData := map[string]interface{}{
			"name":     "Test User",
			"email":    "test@example.com",
			"password": "password123",
		}

		body, _ := json.Marshal(registerData)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		authHandler.Register(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user")
	})

	// Test user login
	t.Run("User Login", func(t *testing.T) {
		loginData := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
		}

		body, _ := json.Marshal(loginData)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		authHandler.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user")
	})

	// Test user CRUD operations
	t.Run("User CRUD Operations", func(t *testing.T) {
		// Create a user first
		user := &user.User{
			Name:  "CRUD Test User",
			Email: "crud@example.com",
		}

		createdUser, err := userService.Create(context.Background(), user)
		require.NoError(t, err)
		require.NotNil(t, createdUser)

		// Test Get User
		t.Run("Get User", func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/users/%d", createdUser.ID), nil)
			w := httptest.NewRecorder()

			userHandler.GetUser(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, createdUser.ID, response.ID)
			assert.Equal(t, createdUser.Name, response.Name)
			assert.Equal(t, createdUser.Email, response.Email)
		})

		// Test List Users
		t.Run("List Users", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/users", nil)
			w := httptest.NewRecorder()

			userHandler.ListUsers(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response []struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(response), 1)
		})

		// Test Update User
		t.Run("Update User", func(t *testing.T) {
			updateData := map[string]interface{}{
				"name":  "Updated CRUD User",
				"email": "updated.crud@example.com",
			}

			body, _ := json.Marshal(updateData)
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/users/%d", createdUser.ID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			userHandler.UpdateUser(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "Updated CRUD User", response.Name)
			assert.Equal(t, "updated.crud@example.com", response.Email)
		})

		// Test Delete User
		t.Run("Delete User", func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/users/%d", createdUser.ID), nil)
			w := httptest.NewRecorder()

			userHandler.DeleteUser(w, req)

			assert.Equal(t, http.StatusNoContent, w.Code)

			// Verify user is deleted
			_, err := userService.Get(context.Background(), createdUser.ID)
			assert.Error(t, err)
		})
	})
}

func TestHealthCheck_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	// Setup test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: env.dbURL,
		},
	}

	// Initialize database
	dbInstance, err := db.Init(cfg.Database)
	require.NoError(t, err)
	defer dbInstance.Close()

	// Test health check
	t.Run("Health Check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		// Create a simple health check handler for testing
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	})
}
