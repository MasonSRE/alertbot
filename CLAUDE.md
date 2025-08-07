# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AlertBot is a modern alert management platform designed to replace Prometheus Alertmanager with a friendly Web UI and powerful alert processing capabilities. It features a Go backend with Gin + PostgreSQL + GORM, and a React 18 + TypeScript + Ant Design + Vite frontend.

## Development Commands

### Environment Setup (Full Docker Stack)
```bash
# Start all services (Database + Backend + Frontend)
docker-compose up -d

# Or start services individually:
# Start PostgreSQL database
docker-compose up -d postgres

# Run database migration (if needed)
go run cmd/migrate/main.go

# Start backend server
docker-compose up -d alertbot

# Start frontend
docker-compose up -d frontend
```

### ⚠️ IMPORTANT: Full Docker Deployment
**ALL services now run in Docker containers for consistency:**

```bash
# ✅ CORRECT: Start all services with Docker
docker-compose up -d

# ✅ CORRECT: Start individual services
docker-compose up -d postgres alertbot frontend

# ❌ WRONG: Do not start services directly
# go run cmd/server/main.go
# cd web && npm run dev
```

**Benefits of Full Docker Stack:**
- Consistent environment across all services
- Proper service networking and dependencies
- Production-like deployment setup
- Unified service management and health checks
- No need for local Node.js or Go installations

### Service Status Verification
After starting all services, verify they're running correctly:

```bash
# Check all container status
docker-compose ps

# Check service health
curl http://localhost:8080/health       # Backend health
curl http://localhost:8080/api/v1/health # API health  
curl http://localhost:3000              # Frontend access

# View service logs
docker-compose logs -f alertbot --tail=20   # Backend logs
docker-compose logs -f frontend --tail=20   # Frontend logs
docker-compose logs -f postgres --tail=20   # Database logs
```

**Expected Service Status:**
- **Database**: `alertbot_postgres` - Port 5432 - Healthy ✅
- **Backend**: `alertbot_server` - Port 8080 - Running ✅
- **Frontend**: `alertbot_frontend` - Port 3000 - Running ✅

**Expected Health Responses:**
- Backend health: `{"service":"alertbot","status":"ok","version":"1.0.0"}` ✅
- API health: `{"success":true,"data":{"status":"healthy",...}}` ✅
- Frontend: AlertBot Web UI accessible at http://localhost:3000 ✅

**🎉 All Services Successfully Running!**
- Frontend UI: http://localhost:3000
- Backend API: http://localhost:8080
- Database: localhost:5432

### Development Scripts
- `./start-dev.sh` - Automated development environment startup script
- `./test-api.sh` - API testing script with comprehensive endpoint coverage

### Build and Test Commands
- **Backend Build**: `go build -o alertbot cmd/server/main.go`
- **Frontend Build**: `cd web && npm run build`
- **Frontend Lint**: `cd web && npm run lint`
- **API Testing**: `./test-api.sh` (requires running server)

### Key API Endpoints
- **Authentication**: `/api/v1/auth/*` - Login, logout, token refresh
- **Alerts**: `/api/v1/alerts/*` - Alert management and operations
- **Rules**: `/api/v1/rules/*` - Routing rule CRUD and testing
- **Channels**: `/api/v1/channels/*` - Notification channel management
- **Silences**: `/api/v1/silences/*` - Silence rule management
- **Statistics**: `/api/v1/stats/*` - Alert and notification statistics
- **Health**: `/api/v1/health` - System health check

### Container Operations
```bash
# Full stack with containers
docker-compose up -d

# View logs
docker-compose logs -f alertbot

# Database operations
docker-compose exec postgres psql -U alertbot -d alertbot
```

## Architecture Overview

### Backend Structure (Go)
- **cmd/**: Entry points (server, migrate)
- **internal/**: Core application logic
  - **api/**: HTTP handlers and routing (alert_handler.go, router.go)
  - **service/**: Business logic layer with interfaces (interfaces.go, alert_service.go)
  - **repository/**: Data access layer with GORM
  - **models/**: Data models with JSONB support for PostgreSQL (models.go)
  - **middleware/**: HTTP middleware (auth, cors, logging, rate limiting)
  - **engine/**: Rule engine for alert processing
  - **notification/**: Multi-channel notification system (dingtalk, wechat_work, email, sms)
  - **websocket/**: Real-time alert streaming
- **pkg/**: Reusable packages (logger, utils)

### Frontend Structure (React + TypeScript)
- **src/components/**: Reusable UI components
- **src/pages/**: Page-level components (Dashboard, Alerts, Rules, Channels, Settings, Test)
- **src/hooks/**: Custom React hooks (useAlerts.ts)
- **src/services/**: API client layer (api.ts)
- **src/stores/**: Zustand state management
- **src/types/**: TypeScript type definitions

### Key Dependencies
**Backend:**
- Gin web framework with CORS and security middleware
- GORM with PostgreSQL driver
- JWT authentication (golang-jwt/jwt)
- WebSocket support (gorilla/websocket)
- Prometheus metrics integration
- Structured logging with logrus

**Frontend:**
- React 18 with TypeScript
- Ant Design UI components
- React Router for navigation
- TanStack React Query for data fetching
- Zustand for state management
- Recharts for visualization
- Axios for HTTP client

## Development Guidelines

### Configuration Management
- Configuration via `configs/config.yaml`
- Environment-specific settings supported
- Database connection pooling configured
- Rate limiting and security settings included

### API Design Patterns
- RESTful API with `/api/v1` prefix
- Prometheus-compatible alert format support
- WebSocket endpoint at `/api/v1/ws/alerts`
- Comprehensive error handling with structured responses

### Database Operations
- PostgreSQL with JSONB support for flexible schema
- GORM for ORM with proper indexing strategy
- Migration system in `cmd/migrate/main.go`
- Alert fingerprinting for deduplication

### Alert Processing Flow
1. Receive alerts via POST `/api/v1/alerts` (Prometheus format)
2. Generate fingerprints for deduplication
3. Process through rule engine (`internal/engine/rule_engine.go`)
4. Route to appropriate notification channels
5. Real-time WebSocket broadcasting
6. Store in PostgreSQL with history tracking

### Notification Channels
- DingTalk integration
- WeChat Work support
- Email notifications
- SMS capabilities
- Configurable via UI with test functionality

### Testing Strategy
- API testing via `test-api.sh` script
- Health check endpoint at `/health`
- Database connectivity verification
- Frontend linting with ESLint + TypeScript

## Recently Implemented Features

### Core Business Logic
- **Standardized API Responses**: Unified response format across all endpoints with proper error handling
- **Complete Notification Channel Management**: Full CRUD operations with validation and testing capabilities for DingTalk, WeChat Work, Email, and SMS channels
- **Advanced Silence Management**: Comprehensive silence rules with regex support, validation, and testing functionality
- **Statistics and Analytics**: Real-time alert and notification statistics with grouping and timeline data
- **Authentication Framework**: Basic authentication system with JWT token support (placeholder implementation)

### Enhanced Error Handling
- **Response Helper**: Centralized response handling with consistent error codes and messages
- **Input Validation**: Comprehensive request validation with detailed error responses
- **Edge Case Handling**: Proper handling of missing data, invalid parameters, and system failures

### API Completeness
- All major CRUD operations fully implemented for alerts, rules, channels, and silences
- Comprehensive testing endpoints for rules and silence matchers
- Advanced filtering and pagination support
- Real-time WebSocket integration for alert updates

## Important Notes

- Server runs on port 8080 (backend) and 3000 (frontend dev)
- Database uses connection pooling (max 100 connections)  
- JWT tokens expire in 24 hours by default
- Rate limiting: 100 requests/minute enabled
- All sensitive configuration should use environment variables in production
- JSONB fields in models support flexible alert metadata storage
- WebSocket hub manages real-time connections for alert streaming
- API responses follow standardized format with success/error indicators
- Notification channel configurations are validated based on channel type
- Statistics provide real-time insights with configurable time ranges and grouping