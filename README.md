# Connex

A comprehensive full-stack web application built with Go backend and modern frontend technologies. Features include real-time WebSocket communication, secure authentication, static file serving, and server-side rendering capabilities.

## 🚀 Features

### Backend (Go)
- **HTTP API** using `net/http` and `chi` router
- **JSON-based REST endpoints** with comprehensive error handling
- **JWT-based authentication** with role support and CSRF protection
- **PostgreSQL integration** using `sqlx` with migrations
- **Redis caching** and session management
- **WebSocket support** with authentication, rate limiting, and room-based messaging
- **Static file serving** with SPA fallback for React Router
- **Server-side rendering hooks** for future SSR implementation
- **Background job processing** with asynq
- **Comprehensive monitoring** with Prometheus, OpenTelemetry, and health checks
- **Security-first approach** with rate limiting, input validation, and security headers

### Frontend
- **Modern responsive UI** with CSS Grid and Flexbox
- **Real-time chat** via WebSocket connections
- **Authentication system** with JWT token management
- **SPA architecture** with client-side routing support
- **Static file serving** from Go backend
- **SSR-ready templates** for future server-side rendering
- **Admin Dashboard** with glassmorphic design and real-time monitoring

### Infrastructure
- **Docker containerization** with multi-stage builds
- **Docker Compose** for development and production
- **Load testing** with k6 and comprehensive test suites
- **Security scanning** with automated vulnerability detection
- **CI/CD ready** with comprehensive testing and deployment scripts

## 🏗️ Project Structure

```
connex/
├── cmd/                    # Application entry points
│   └── server/
│       └── main.go        # Main server with WebSocket and static file support
├── internal/              # Application logic
│   ├── api/              # HTTP handlers and WebSocket
│   │   ├── auth/         # Authentication handlers
│   │   ├── user/         # User management
│   │   ├── websocket/    # WebSocket handler with rooms and messaging
│   │   └── ssr/          # Server-side rendering hooks
│   ├── service/          # Business logic
│   ├── db/               # Database access & migrations
│   ├── middleware/       # HTTP middleware (security, logging, etc.)
│   ├── job/              # Background tasks
│   └── config/           # Configuration management
├── pkg/                  # Shared libraries
│   ├── hash/             # Password hashing
│   ├── jwt/              # JWT utilities
│   └── logger/           # Structured logging
├── web/                  # Frontend application
│   ├── public/           # Static assets (served by Go)
│   │   ├── index.html    # Main SPA with WebSocket chat
│   │   └── admin.html    # Admin dashboard with glassmorphic UI
│   └── src/              # Frontend source code
├── tests/                # Comprehensive test suites
├── scripts/              # Build and deployment scripts
├── Dockerfile            # Multi-stage container build
├── docker-compose.yml    # Development environment
├── docker-compose.prod.yml # Production environment
├── Makefile              # Build automation
└── README.md             # This file
```

## 🛠️ Requirements

- **Go** >= 1.24.3
- **PostgreSQL** >= 14
- **Redis** >= 6
- **Docker** (optional, for containerized setup)
- **Node.js** >= 18 (for frontend development)

## 🚀 Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/wdarrenww/connex.git
cd connex
```

### 2. Environment Configuration

Create a `.env` file based on `env.example`:

```bash
cp env.example .env
```

Configure your environment variables:

```env
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgres://user:password@localhost:5432/connex?sslmode=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-super-secret-jwt-key-32-chars-minimum

# CSRF (base64-encoded 32-byte key)
CSRF_AUTH_KEY=your-base64-encoded-32-byte-csrf-key

# OpenTelemetry
OTEL_ENABLED=true
OTEL_ENDPOINT=http://localhost:14268/api/traces
```

### 3. Install Dependencies

```bash
# Backend dependencies
go mod tidy

# Frontend dependencies (if developing frontend)
cd web
npm install
```

### 4. Start Services

#### Option A: Docker Compose (Recommended)

```bash
# Start all services (PostgreSQL, Redis, Jaeger, Prometheus, Grafana)
make dev-docker

# In another terminal, start the Go application
make run
```

#### Option B: Manual Setup

```bash
# Start PostgreSQL and Redis manually
# Then run the application
make run
```

### 5. Access the Application

- **Web Application**: http://localhost:8080
- **Admin Dashboard**: http://localhost:8080/admin
- **API Documentation**: http://localhost:8080/api/health
- **Metrics**: http://localhost:8080/metrics
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686

## 🔌 WebSocket API

The application includes a comprehensive WebSocket implementation at `/ws`:

### Connection

```javascript
// Connect with JWT authentication
const ws = new WebSocket(`ws://localhost:8080/ws?token=${jwtToken}`);
```

### Message Types

```javascript
// Chat message
{
  "type": "chat",
  "data": "Hello, world!",
  "timestamp": "2025-07-03T21:58:45.123Z"
}

// Join room
{
  "type": "auth",
  "data": {
    "room": "general"
  }
}

// Ping/Pong (automatic)
{
  "type": "ping",
  "data": {},
  "timestamp": "2025-07-03T21:58:45.123Z"
}
```

### Features

- **Authentication**: JWT token validation
- **Rate Limiting**: 10 connections per minute per IP
- **Room Support**: Join/leave chat rooms
- **Message Broadcasting**: Send to all clients or specific rooms
- **Automatic Ping/Pong**: Connection health monitoring
- **Error Handling**: Comprehensive error responses

## 📁 Static File Serving

The application serves static files from `web/public/`:

- **Static Assets**: `/static/*` - CSS, JS, images, etc.
- **SPA Fallback**: Any unknown route serves `index.html` for React Router
- **Security**: Proper cache headers and security middleware

### Frontend Build

```bash
# Build frontend (if using a build tool)
cd web
npm run build

# Copy build output to public directory
cp -r dist/* public/
```

## 🔐 Security Features

### Authentication & Authorization
- JWT-based authentication with secure token handling
- Password hashing with bcrypt
- CSRF protection on state-changing requests
- Role-based access control

### Input Validation & Sanitization
- Comprehensive input validation for all endpoints
- XSS protection with content filtering
- SQL injection prevention
- Request size limiting (1MB default)

### Security Headers
- Content Security Policy (CSP)
- X-Content-Type-Options
- X-Frame-Options
- X-XSS-Protection
- Modern security headers (COEP, COOP, etc.)

### Rate Limiting
- IP-based rate limiting for authentication endpoints
- WebSocket connection rate limiting
- Configurable limits and time windows

## 🧪 Testing

### Run All Tests

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Security tests
make security-test-comprehensive

# Load tests
make load-test
```

### Test Coverage

```bash
make test-coverage
```

## 📊 Monitoring & Observability

### Metrics
- Prometheus metrics at `/metrics`
- Custom application metrics
- Database and Redis monitoring

### Tracing
- OpenTelemetry integration
- Jaeger for distributed tracing
- Request tracing middleware

### Health Checks
- `/health` - Basic health check
- `/health/detailed` - Comprehensive health status
- `/ready` - Readiness probe

## 🐳 Docker Deployment

### Development

```bash
docker-compose up --build
```

### Production

```bash
docker-compose -f docker-compose.prod.yml up --build
```

### Production Features
- Multi-stage builds for smaller images
- Security hardening
- Resource limits
- Health checks
- Graceful shutdown

## 🔧 Development

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Run security scans
make security-all
```

### Database Migrations

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down
```

### Background Jobs

```bash
# Start job worker
go run ./cmd/worker

# Enqueue jobs via API
curl -X POST http://localhost:8080/api/jobs/email
```

## 🚀 Production Deployment

### Environment Variables

Ensure all production environment variables are set:

```bash
# Required for production
ENV=production
JWT_SECRET=<secure-32-char-minimum>
CSRF_AUTH_KEY=<base64-encoded-32-byte-key>
DATABASE_URL=<production-database-url>
REDIS_PASSWORD=<redis-password>
```

### Security Checklist

- [ ] Change default passwords
- [ ] Configure HTTPS/TLS
- [ ] Set up proper CORS origins
- [ ] Configure rate limiting for production
- [ ] Set up monitoring and alerting
- [ ] Regular security scans
- [ ] Database backups
- [ ] Log aggregation

## 📚 API Endpoints

### Authentication
- `POST /api/auth/register` - User registration
- `POST /api/auth/login` - User login

### Users
- `GET /api/users/me` - Get current user
- `PUT /api/users/me` - Update current user
- `DELETE /api/users/me` - Delete current user

### Admin (Protected)
- `GET /api/admin/dashboard` - Dashboard overview data
- `GET /api/admin/users` - User management data
- `GET /api/admin/analytics` - Analytics and reporting
- `GET /api/admin/system` - System status and health
- `GET /api/admin/logs` - System logs
- `GET /api/admin/metrics` - System metrics

### Health & Monitoring
- `GET /health` - Basic health check
- `GET /health/detailed` - Detailed health status
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### WebSocket
- `GET /ws` - WebSocket endpoint

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For support and questions:
- Create an issue on GitHub
- Check the documentation
- Review the security audit report

---

**Built with ❤️ using Go, WebSockets, and modern web technologies**
