package db

import (
	"fmt"
	"sync"

	"connex/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	db   *sqlx.DB
	once sync.Once
)

// Init initializes the database connection pool
func Init(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	var err error
	once.Do(func() {
		if cfg.URL != "" {
			db, err = sqlx.Connect("postgres", cfg.URL)
		} else {
			dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
				cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
			db, err = sqlx.Connect("postgres", dsn)
		}
	})
	return db, err
}

// Get returns the global DB instance
func Get() *sqlx.DB {
	return db
}
