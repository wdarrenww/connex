# Connex

This project is a full-stack web application built with a Go-based backend and a JavaScript frontend (React). It includes a RESTful API, authentication, PostgreSQL integration, background jobs, and frontend integration via static file serving or reverse proxy.

## Features

### Backend (Go)
- HTTP API using `net/http` and `chi` router
- JSON-based REST endpoints
- JWT-based authentication with role support
- PostgreSQL integration using `sqlx` or `gorm`
- Configurable via `.env` files
- Graceful shutdown support
- Middleware: logging, recovery, CORS, request ID
- Background job runner (e.g., for sending emails)
- Structured logging with zap
- Health check and readiness endpoints
- Optional WebSocket support

### Frontend (React or similar)
- SPA (Single Page Application)
- Built assets can be served via Go backend or hosted separately
- Authentication-aware routes
- Axios-based API integration
- Responsive layout

## Project Structure

```
connex/
├── cmd/                # Entry point
│   └── server/
│       └── main.go
├── internal/           # Application logic
│   ├── api/            # HTTP handlers
│   ├── service/        # Business logic
│   ├── db/             # Data access & migrations
│   ├── middleware/     # HTTP middleware
│   ├── job/            # Background tasks
│   └── config/         # App configuration
├── pkg/                # Utility libraries (JWT, logger, etc.)
├── web/                # Frontend application
├── tests/              # Integration and e2e tests
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env
└── go.mod
```

## Requirements

- Go >= 1.21
- Node.js >= 18 (for frontend)
- PostgreSQL >= 14
- Docker (optional for local setup)

## Setup Instructions

### 1. Clone the Repository

```
git clone https://github.com/yourusername/your-app.git
cd your-app
```

### 2. Backend Setup

#### Environment Configuration

Create a `.env` file in the root:

```
PORT=8080
ENV=development
DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
JWT_SECRET=supersecurekey
```

#### Install Dependencies

```
go mod tidy
```

#### Run the Server

```
go run ./cmd/server
```

### 3. Database

Run migrations:

```
make migrate-up
```

To rollback:

```
make migrate-down
```

Or use a migration tool like `golang-migrate`.

### 4. Frontend Setup

```
cd web
npm install
npm run dev
```

To build for production:

```
npm run build
```

To serve via Go backend, copy `dist/` into a folder served by your Go static file handler.

### 5. Run All Services with Docker Compose

```
docker-compose up --build
```

## Development Tools

- **Go HTTP Router**: [chi](https://github.com/go-chi/chi)
- **PostgreSQL ORM**: `sqlx` or `gorm`
- **JWT Auth**: `github.com/golang-jwt/jwt`
- **Frontend**: React + Vite
- **Logging**: `uber-go/zap`
- **Testing**: `testing`, `httptest`, `testcontainers-go`
- **Background Jobs**: `asynq` (optional)
- **Task Automation**: `Makefile`

## API Endpoints (Examples)

- `POST /api/auth/login`
- `POST /api/auth/register`
- `GET /api/users/me`
- `GET /api/health`
- `POST /api/tasks/send-email`

## Testing

Run unit tests:

```
go test ./...
```

Run integration tests:

```
go test -tags=integration ./tests
```

## Deployment

### Build the Go binary

```
make build
```

### Build Docker image

```
docker build -t your-app .
```

### Deploy to Fly.io / Railway / GCP / AWS

Recommended: build frontend separately and serve via CDN; deploy backend as container.

## License

This project is licensed under the MIT License.
