package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connex/internal/config"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

var (
	client *asynq.Client
	server *asynq.Server
	logger *zap.Logger
)

// Job types
const (
	TypeEmailSend   = "email:send"
	TypeDataProcess = "data:process"
	TypeUserWelcome = "user:welcome"
	TypeCleanup     = "cleanup"
)

// EmailPayload represents email job data
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// UserWelcomePayload represents user welcome job data
type UserWelcomePayload struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
}

// DataProcessPayload represents data processing job data
type DataProcessPayload struct {
	DataID   string `json:"data_id"`
	Process  string `json:"process"`
	Priority int    `json:"priority"`
}

// Init initializes the job queue system
func Init(cfg config.JobsConfig, redisOpt asynq.RedisClientOpt, log *zap.Logger) error {
	logger = log

	// Create client
	client = asynq.NewClient(redisOpt)

	// Create server
	server = asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues: map[string]int{
				"critical": 10,
				"default":  5,
				"low":      1,
			},
		},
	)

	// Register handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeEmailSend, handleEmailSend)
	mux.HandleFunc(TypeUserWelcome, handleUserWelcome)
	mux.HandleFunc(TypeDataProcess, handleDataProcess)
	mux.HandleFunc(TypeCleanup, handleCleanup)

	// Start server in background
	go func() {
		if err := server.Run(mux); err != nil {
			logger.Error("failed to run job server", zap.Error(err))
		}
	}()

	return nil
}

// GetClient returns the job client
func GetClient() *asynq.Client {
	return client
}

// GetServer returns the job server
func GetServer() *asynq.Server {
	return server
}

// EnqueueEmail enqueues an email job
func EnqueueEmail(payload EmailPayload, opts ...asynq.Option) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeEmailSend, data)
	_, err = client.Enqueue(task, opts...)
	return err
}

// EnqueueUserWelcome enqueues a user welcome job
func EnqueueUserWelcome(payload UserWelcomePayload, opts ...asynq.Option) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeUserWelcome, data)
	_, err = client.Enqueue(task, opts...)
	return err
}

// EnqueueDataProcess enqueues a data processing job
func EnqueueDataProcess(payload DataProcessPayload, opts ...asynq.Option) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeDataProcess, data)
	_, err = client.Enqueue(task, opts...)
	return err
}

// EnqueueCleanup enqueues a cleanup job
func EnqueueCleanup(opts ...asynq.Option) error {
	task := asynq.NewTask(TypeCleanup, nil)
	_, err := client.Enqueue(task, opts...)
	return err
}

// Job handlers
func handleEmailSend(ctx context.Context, t *asynq.Task) error {
	var payload EmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal email payload: %w", err)
	}

	logger.Info("sending email",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
	)

	// Simulate email sending
	time.Sleep(100 * time.Millisecond)
	logger.Info("email sent successfully", zap.String("to", payload.To))

	return nil
}

func handleUserWelcome(ctx context.Context, t *asynq.Task) error {
	var payload UserWelcomePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user welcome payload: %w", err)
	}

	logger.Info("sending welcome email",
		zap.Int64("user_id", payload.UserID),
		zap.String("email", payload.Email),
	)

	// Simulate welcome email
	time.Sleep(50 * time.Millisecond)
	logger.Info("welcome email sent", zap.Int64("user_id", payload.UserID))

	return nil
}

func handleDataProcess(ctx context.Context, t *asynq.Task) error {
	var payload DataProcessPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal data process payload: %w", err)
	}

	logger.Info("processing data",
		zap.String("data_id", payload.DataID),
		zap.String("process", payload.Process),
		zap.Int("priority", payload.Priority),
	)

	// Simulate data processing
	time.Sleep(200 * time.Millisecond)
	logger.Info("data processing completed", zap.String("data_id", payload.DataID))

	return nil
}

func handleCleanup(ctx context.Context, t *asynq.Task) error {
	logger.Info("running cleanup job")

	// Simulate cleanup
	time.Sleep(100 * time.Millisecond)
	logger.Info("cleanup completed")

	return nil
}

// asynqLogger implements asynq.Logger interface
type asynqLogger struct {
	logger *zap.Logger
}

func (l *asynqLogger) Debug(msg string, fields map[string]interface{}) {
	l.logger.Debug(msg, zap.Any("fields", fields))
}

func (l *asynqLogger) Info(msg string, fields map[string]interface{}) {
	l.logger.Info(msg, zap.Any("fields", fields))
}

func (l *asynqLogger) Warn(msg string, fields map[string]interface{}) {
	l.logger.Warn(msg, zap.Any("fields", fields))
}

func (l *asynqLogger) Error(msg string, fields map[string]interface{}) {
	l.logger.Error(msg, zap.Any("fields", fields))
}
