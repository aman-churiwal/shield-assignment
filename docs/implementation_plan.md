# User Analytics Service — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go backend service that tracks user logins and provides daily/monthly unique user counts via REST API, backed by PostgreSQL and packaged with Docker Compose.

**Architecture:** Classic layered architecture — Gin HTTP handlers → Service (business logic, timezone handling) → Repository (raw SQL via database/sql + pgx) → PostgreSQL. Each layer communicates via Go interfaces for testability.

**Tech Stack:** Go 1.22+, Gin, database/sql + pgx, PostgreSQL 16, Docker Compose, Go testing package + testify

## Global Constraints

- Go 1.22 or later
- PostgreSQL 16
- No ORM — use `database/sql` + `pgx` driver for all database access
- All timestamps stored as `TIMESTAMPTZ` (UTC internally)
- RESTful JSON API
- Unit tests for every layer; integration tests for repository
- Modular project structure under `cmd/` and `internal/`

---

### Task 1: Project Scaffolding & Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go` (placeholder entry point)
- Create: `Makefile`

**Interfaces:**
- Consumes: nothing
- Produces: Go module `github.com/user/shield-assignment`, `make run` / `make test` targets

- [ ] **Step 1: Initialize Go module**

```bash
cd d:\Projects\shield-assignment
go mod init github.com/user/shield-assignment
```

- [ ] **Step 2: Create minimal main.go**

Create `cmd/server/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("User Analytics Service starting...")
}
```

- [ ] **Step 3: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: run test build

run:
	go run ./cmd/server

test:
	go test ./... -v -count=1

build:
	go build -o bin/server ./cmd/server

lint:
	go vet ./...
```

- [ ] **Step 4: Verify scaffolding**

Run: `go run ./cmd/server`
Expected: prints "User Analytics Service starting..."

- [ ] **Step 5: Commit**

```bash
git init
git add .
git commit -m "chore: initialize Go module and project scaffolding"
```

---

### Task 2: Domain Models

**Files:**
- Create: `internal/model/model.go`

**Interfaces:**
- Consumes: nothing
- Produces: `model.LoginEvent`, `model.LoginRequest`, `model.DailyUniqueUsersResponse`, `model.MonthlyUniqueUsersResponse`, `model.HealthResponse`, `model.ErrorResponse`

- [ ] **Step 1: Create model types**

Create `internal/model/model.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

// LoginEvent represents a user login stored in the database.
type LoginEvent struct {
	ID        int64     `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	LoginTime time.Time `json:"login_time"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest represents the API request to record a login.
type LoginRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	LoginTime string `json:"login_time" binding:"required"`
}

// DailyUniqueUsersResponse is the API response for daily unique user count.
type DailyUniqueUsersResponse struct {
	Date        string `json:"date"`
	Timezone    string `json:"timezone"`
	UniqueUsers int    `json:"unique_users"`
}

// MonthlyUniqueUsersResponse is the API response for monthly unique user count.
type MonthlyUniqueUsersResponse struct {
	Month       string `json:"month"`
	Timezone    string `json:"timezone"`
	UniqueUsers int    `json:"unique_users"`
}

// HealthResponse is the API response for health checks.
type HealthResponse struct {
	Status string `json:"status"`
}

// ErrorResponse is the standard API error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/google/uuid
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/model/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: add domain model types"
```

---

### Task 3: Configuration

**Files:**
- Create: `internal/config/config.go`

**Interfaces:**
- Consumes: nothing
- Produces: `config.Config` struct, `config.Load() (*Config, error)`

- [ ] **Step 1: Create config loader**

Create `internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ServerPort string
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "analytics"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/config/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add .
git commit -m "feat: add environment-based configuration"
```

---

### Task 4: Database Migration

**Files:**
- Create: `migrations/001_create_user_logins.sql`

**Interfaces:**
- Consumes: nothing
- Produces: SQL migration file applied by repository or main.go at startup

- [ ] **Step 1: Create migration SQL**

Create `migrations/001_create_user_logins.sql`:
```sql
CREATE TABLE IF NOT EXISTS user_logins (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL,
    login_time  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent exact duplicate logins (same user, same timestamp)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_logins_unique
    ON user_logins (user_id, login_time);

-- Speed up time-range aggregation queries
CREATE INDEX IF NOT EXISTS idx_user_logins_login_time
    ON user_logins (login_time);
```

- [ ] **Step 2: Commit**

```bash
git add .
git commit -m "feat: add database migration for user_logins table"
```

---

### Task 5: Repository Layer (Interface + PostgreSQL Implementation)

**Files:**
- Create: `internal/repository/repository.go`

**Interfaces:**
- Consumes: `model.LoginEvent`, `config.Config`
- Produces:
  ```go
  type Repository interface {
      InsertLogin(ctx context.Context, userID uuid.UUID, loginTime time.Time) error
      GetDailyUniqueUsers(ctx context.Context, date time.Time, timezone string) (int, error)
      GetMonthlyUniqueUsers(ctx context.Context, year int, month int, timezone string) (int, error)
      RunMigrations(ctx context.Context) error
      Close() error
  }
  func NewPostgresRepository(db *sql.DB) *PostgresRepository
  ```

- [ ] **Step 1: Write the repository interface and implementation**

Create `internal/repository/repository.go`:
```go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Repository defines the data access interface.
type Repository interface {
	// InsertLogin records a login event. Returns nil if duplicate.
	InsertLogin(ctx context.Context, userID uuid.UUID, loginTime time.Time) error
	// GetDailyUniqueUsers returns the count of unique users for a given date in the given timezone.
	GetDailyUniqueUsers(ctx context.Context, date time.Time, timezone string) (int, error)
	// GetMonthlyUniqueUsers returns the count of unique users for a given year/month in the given timezone.
	GetMonthlyUniqueUsers(ctx context.Context, year int, month int, timezone string) (int, error)
	// RunMigrations applies database schema migrations.
	RunMigrations(ctx context.Context) error
	// Close closes the database connection.
	Close() error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ConnectDB opens a database connection.
func ConnectDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func (r *PostgresRepository) InsertLogin(ctx context.Context, userID uuid.UUID, loginTime time.Time) error {
	query := `
		INSERT INTO user_logins (user_id, login_time)
		VALUES ($1, $2)
		ON CONFLICT (user_id, login_time) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, userID, loginTime.UTC())
	if err != nil {
		return fmt.Errorf("failed to insert login: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetDailyUniqueUsers(ctx context.Context, date time.Time, timezone string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM user_logins
		WHERE login_time >= $1::date AT TIME ZONE $2
		  AND login_time < ($1::date + INTERVAL '1 day') AT TIME ZONE $2
	`
	dateStr := date.Format("2006-01-02")
	var count int
	err := r.db.QueryRowContext(ctx, query, dateStr, timezone).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily unique users: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) GetMonthlyUniqueUsers(ctx context.Context, year int, month int, timezone string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM user_logins
		WHERE login_time >= DATE_TRUNC('month', $1::date) AT TIME ZONE $2
		  AND login_time < (DATE_TRUNC('month', $1::date) + INTERVAL '1 month') AT TIME ZONE $2
	`
	dateStr := fmt.Sprintf("%04d-%02d-01", year, month)
	var count int
	err := r.db.QueryRowContext(ctx, query, dateStr, timezone).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get monthly unique users: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) RunMigrations(ctx context.Context) error {
	migration := `
		CREATE TABLE IF NOT EXISTS user_logins (
			id          BIGSERIAL PRIMARY KEY,
			user_id     UUID NOT NULL,
			login_time  TIMESTAMPTZ NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_logins_unique
			ON user_logins (user_id, login_time);
		CREATE INDEX IF NOT EXISTS idx_user_logins_login_time
			ON user_logins (login_time);
	`
	_, err := r.db.ExecContext(ctx, migration)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
```

- [ ] **Step 2: Add pgx dependency**

```bash
go get github.com/jackc/pgx/v5
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/repository/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: add repository layer with PostgreSQL implementation"
```

---

### Task 6: Service Layer

**Files:**
- Create: `internal/service/service.go`
- Create: `internal/service/service_test.go`

**Interfaces:**
- Consumes: `repository.Repository` interface
- Produces:
  ```go
  type AnalyticsService interface {
      RecordLogin(ctx context.Context, userID string, loginTime string) error
      GetDailyUniqueUsers(ctx context.Context, date string, timezone string) (int, error)
      GetMonthlyUniqueUsers(ctx context.Context, month string, timezone string) (int, error)
  }
  func NewAnalyticsService(repo repository.Repository) *analyticsService
  ```

- [ ] **Step 1: Write failing tests for the service**

Create `internal/service/service_test.go`:
```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository implements repository.Repository for testing.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) InsertLogin(ctx context.Context, userID uuid.UUID, loginTime time.Time) error {
	args := m.Called(ctx, userID, loginTime)
	return args.Error(0)
}

func (m *MockRepository) GetDailyUniqueUsers(ctx context.Context, date time.Time, timezone string) (int, error) {
	args := m.Called(ctx, date, timezone)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetMonthlyUniqueUsers(ctx context.Context, year int, month int, timezone string) (int, error) {
	args := m.Called(ctx, year, month, timezone)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) RunMigrations(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRecordLogin_ValidInput(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	userID := uuid.New().String()
	loginTime := "2024-01-15T14:30:00Z"

	parsedID, _ := uuid.Parse(userID)
	parsedTime, _ := time.Parse(time.RFC3339, loginTime)

	mockRepo.On("InsertLogin", mock.Anything, parsedID, parsedTime.UTC()).Return(nil)

	err := svc.RecordLogin(context.Background(), userID, loginTime)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRecordLogin_InvalidUUID(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	err := svc.RecordLogin(context.Background(), "not-a-uuid", "2024-01-15T14:30:00Z")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
}

func TestRecordLogin_InvalidTime(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	err := svc.RecordLogin(context.Background(), uuid.New().String(), "not-a-time")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid login_time")
}

func TestGetDailyUniqueUsers_ValidInput(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	expectedDate, _ := time.Parse("2006-01-02", "2024-01-15")
	mockRepo.On("GetDailyUniqueUsers", mock.Anything, expectedDate, "UTC").Return(42, nil)

	count, err := svc.GetDailyUniqueUsers(context.Background(), "2024-01-15", "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 42, count)
	mockRepo.AssertExpectations(t)
}

func TestGetDailyUniqueUsers_InvalidDate(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	_, err := svc.GetDailyUniqueUsers(context.Background(), "invalid", "UTC")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestGetDailyUniqueUsers_InvalidTimezone(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	_, err := svc.GetDailyUniqueUsers(context.Background(), "2024-01-15", "Not/A/Timezone")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timezone")
}

func TestGetDailyUniqueUsers_DefaultTimezone(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	expectedDate, _ := time.Parse("2006-01-02", "2024-01-15")
	mockRepo.On("GetDailyUniqueUsers", mock.Anything, expectedDate, "UTC").Return(10, nil)

	count, err := svc.GetDailyUniqueUsers(context.Background(), "2024-01-15", "")
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestGetMonthlyUniqueUsers_ValidInput(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	mockRepo.On("GetMonthlyUniqueUsers", mock.Anything, 2024, 1, "UTC").Return(150, nil)

	count, err := svc.GetMonthlyUniqueUsers(context.Background(), "2024-01", "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 150, count)
	mockRepo.AssertExpectations(t)
}

func TestGetMonthlyUniqueUsers_InvalidMonth(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	_, err := svc.GetMonthlyUniqueUsers(context.Background(), "invalid", "UTC")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month format")
}

func TestGetDailyUniqueUsers_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	expectedDate, _ := time.Parse("2006-01-02", "2024-01-15")
	mockRepo.On("GetDailyUniqueUsers", mock.Anything, expectedDate, "UTC").Return(0, errors.New("db error"))

	_, err := svc.GetDailyUniqueUsers(context.Background(), "2024-01-15", "UTC")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -v -count=1`
Expected: FAIL — `NewAnalyticsService` undefined

- [ ] **Step 3: Implement the service**

Create `internal/service/service.go`:
```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/user/shield-assignment/internal/repository"
)

// AnalyticsService defines the business logic interface.
type AnalyticsService interface {
	RecordLogin(ctx context.Context, userID string, loginTime string) error
	GetDailyUniqueUsers(ctx context.Context, date string, timezone string) (int, error)
	GetMonthlyUniqueUsers(ctx context.Context, month string, timezone string) (int, error)
}

type analyticsService struct {
	repo repository.Repository
}

// NewAnalyticsService creates a new AnalyticsService.
func NewAnalyticsService(repo repository.Repository) AnalyticsService {
	return &analyticsService{repo: repo}
}

func (s *analyticsService) RecordLogin(ctx context.Context, userID string, loginTime string) error {
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	parsedTime, err := time.Parse(time.RFC3339, loginTime)
	if err != nil {
		return fmt.Errorf("invalid login_time: %w", err)
	}

	return s.repo.InsertLogin(ctx, parsedID, parsedTime.UTC())
}

func (s *analyticsService) GetDailyUniqueUsers(ctx context.Context, date string, timezone string) (int, error) {
	if timezone == "" {
		timezone = "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return 0, fmt.Errorf("invalid timezone: %w", err)
	}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	return s.repo.GetDailyUniqueUsers(ctx, parsedDate, timezone)
}

func (s *analyticsService) GetMonthlyUniqueUsers(ctx context.Context, month string, timezone string) (int, error) {
	if timezone == "" {
		timezone = "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return 0, fmt.Errorf("invalid timezone: %w", err)
	}

	parsedDate, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, fmt.Errorf("invalid month format, expected YYYY-MM: %w", err)
	}

	return s.repo.GetMonthlyUniqueUsers(ctx, parsedDate.Year(), int(parsedDate.Month()), timezone)
}
```

- [ ] **Step 4: Add testify dependency and run tests**

```bash
go get github.com/stretchr/testify
go test ./internal/service/ -v -count=1
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: add service layer with business logic and unit tests"
```

---

### Task 7: HTTP Handlers

**Files:**
- Create: `internal/handler/handler.go`
- Create: `internal/handler/handler_test.go`

**Interfaces:**
- Consumes: `service.AnalyticsService` interface
- Produces:
  ```go
  type Handler struct { ... }
  func NewHandler(svc service.AnalyticsService) *Handler
  func (h *Handler) RegisterRoutes(r *gin.Engine)
  ```

- [ ] **Step 1: Write failing handler tests**

Create `internal/handler/handler_test.go`:
```go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/shield-assignment/internal/model"
)

// MockService implements service.AnalyticsService for testing.
type MockService struct {
	mock.Mock
}

func (m *MockService) RecordLogin(ctx context.Context, userID string, loginTime string) error {
	args := m.Called(ctx, userID, loginTime)
	return args.Error(0)
}

func (m *MockService) GetDailyUniqueUsers(ctx context.Context, date string, timezone string) (int, error) {
	args := m.Called(ctx, date, timezone)
	return args.Int(0), args.Error(1)
}

func (m *MockService) GetMonthlyUniqueUsers(ctx context.Context, month string, timezone string) (int, error) {
	args := m.Called(ctx, month, timezone)
	return args.Int(0), args.Error(1)
}

func setupRouter(mockSvc *MockService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(mockSvc)
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

func TestHealthCheck(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestRecordLogin_Success(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	body := `{"user_id":"550e8400-e29b-41d4-a716-446655440000","login_time":"2024-01-15T14:30:00Z"}`
	mockSvc.On("RecordLogin", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "2024-01-15T14:30:00Z").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/logins", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRecordLogin_MissingFields(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	body := `{"user_id":"550e8400-e29b-41d4-a716-446655440000"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/logins", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordLogin_ServiceError(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	body := `{"user_id":"bad-uuid","login_time":"2024-01-15T14:30:00Z"}`
	mockSvc.On("RecordLogin", mock.Anything, "bad-uuid", "2024-01-15T14:30:00Z").Return(errors.New("invalid user_id"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/logins", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDailyUniqueUsers_Success(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	mockSvc.On("GetDailyUniqueUsers", mock.Anything, "2024-01-15", "UTC").Return(42, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/analytics/daily?date=2024-01-15&tz=UTC", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.DailyUniqueUsersResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "2024-01-15", resp.Date)
	assert.Equal(t, "UTC", resp.Timezone)
	assert.Equal(t, 42, resp.UniqueUsers)
}

func TestGetDailyUniqueUsers_MissingDate(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/analytics/daily", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDailyUniqueUsers_DefaultTimezone(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	mockSvc.On("GetDailyUniqueUsers", mock.Anything, "2024-01-15", "").Return(10, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/analytics/daily?date=2024-01-15", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetMonthlyUniqueUsers_Success(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	mockSvc.On("GetMonthlyUniqueUsers", mock.Anything, "2024-01", "UTC").Return(150, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/analytics/monthly?month=2024-01&tz=UTC", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.MonthlyUniqueUsersResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "2024-01", resp.Month)
	assert.Equal(t, "UTC", resp.Timezone)
	assert.Equal(t, 150, resp.UniqueUsers)
}

func TestGetMonthlyUniqueUsers_MissingMonth(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/analytics/monthly", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDailyUniqueUsers_ServiceError(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)

	mockSvc.On("GetDailyUniqueUsers", mock.Anything, "2024-01-15", "UTC").Return(0, errors.New("db error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/analytics/daily?date=2024-01-15&tz=UTC", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/handler/ -v -count=1`
Expected: FAIL — `NewHandler` undefined

- [ ] **Step 3: Implement the handlers**

Create `internal/handler/handler.go`:
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user/shield-assignment/internal/model"
	"github.com/user/shield-assignment/internal/service"
)

// Handler holds HTTP handler dependencies.
type Handler struct {
	svc service.AnalyticsService
}

// NewHandler creates a new Handler.
func NewHandler(svc service.AnalyticsService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes sets up all HTTP routes.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
	api := r.Group("/api")
	{
		api.POST("/logins", h.RecordLogin)
		api.GET("/analytics/daily", h.GetDailyUniqueUsers)
		api.GET("/analytics/monthly", h.GetMonthlyUniqueUsers)
	}
}

// Health handles GET /health.
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{Status: "ok"})
}

// RecordLogin handles POST /api/logins.
func (h *Handler) RecordLogin(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	if err := h.svc.RecordLogin(c.Request.Context(), req.UserID, req.LoginTime); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "login recorded"})
}

// GetDailyUniqueUsers handles GET /api/analytics/daily.
func (h *Handler) GetDailyUniqueUsers(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "missing required query parameter: date"})
		return
	}

	tz := c.Query("tz")

	count, err := h.svc.GetDailyUniqueUsers(c.Request.Context(), date, tz)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	timezone := tz
	if timezone == "" {
		timezone = "UTC"
	}

	c.JSON(http.StatusOK, model.DailyUniqueUsersResponse{
		Date:        date,
		Timezone:    timezone,
		UniqueUsers: count,
	})
}

// GetMonthlyUniqueUsers handles GET /api/analytics/monthly.
func (h *Handler) GetMonthlyUniqueUsers(c *gin.Context) {
	month := c.Query("month")
	if month == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "missing required query parameter: month"})
		return
	}

	tz := c.Query("tz")

	count, err := h.svc.GetMonthlyUniqueUsers(c.Request.Context(), month, tz)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	timezone := tz
	if timezone == "" {
		timezone = "UTC"
	}

	c.JSON(http.StatusOK, model.MonthlyUniqueUsersResponse{
		Month:       month,
		Timezone:    timezone,
		UniqueUsers: count,
	})
}
```

- [ ] **Step 4: Add Gin dependency and run tests**

```bash
go get github.com/gin-gonic/gin
go test ./internal/handler/ -v -count=1
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: add HTTP handlers with Gin and handler tests"
```

---

### Task 8: Main Entry Point (Wire Everything Together)

**Files:**
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `config.Load()`, `repository.ConnectDB()`, `repository.NewPostgresRepository()`, `service.NewAnalyticsService()`, `handler.NewHandler()`
- Produces: Running HTTP server on configured port

- [ ] **Step 1: Implement main.go**

Overwrite `cmd/server/main.go`:
```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user/shield-assignment/internal/config"
	"github.com/user/shield-assignment/internal/handler"
	"github.com/user/shield-assignment/internal/repository"
	"github.com/user/shield-assignment/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := repository.ConnectDB(cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create repository and run migrations
	repo := repository.NewPostgresRepository(db)
	if err := repo.RunMigrations(context.Background()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations applied successfully")

	// Create service and handler
	svc := service.NewAnalyticsService(repo)
	h := handler.NewHandler(svc)

	// Setup Gin router
	router := gin.Default()
	h.RegisterRoutes(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := repo.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server exited cleanly")
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/server/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add .
git commit -m "feat: wire up main entry point with graceful shutdown"
```

---

### Task 9: Docker Setup

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`

**Interfaces:**
- Consumes: Go source code, migrations
- Produces: `docker-compose up` brings up PostgreSQL + Go service

- [ ] **Step 1: Create Dockerfile (multi-stage build)**

Create `Dockerfile`:
```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /server .
COPY migrations/ ./migrations/

EXPOSE 8080

CMD ["./server"]
```

- [ ] **Step 2: Create .dockerignore**

Create `.dockerignore`:
```
bin/
.git/
.agent/
docs/
*.md
```

- [ ] **Step 3: Create docker-compose.yml**

Create `docker-compose.yml`:
```yaml
version: "3.8"

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: analytics
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: db
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: analytics
      SERVER_PORT: "8080"
    depends_on:
      db:
        condition: service_healthy

volumes:
  pgdata:
```

- [ ] **Step 4: Verify docker-compose config**

Run: `docker-compose config`
Expected: valid YAML output, no errors

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: add Docker and docker-compose setup"
```

---

### Task 10: Integration Tests (Repository Layer)

**Files:**
- Create: `internal/repository/repository_test.go`

**Interfaces:**
- Consumes: `repository.PostgresRepository`, a running PostgreSQL instance
- Produces: Integration tests verifying actual SQL behavior

- [ ] **Step 1: Write integration tests**

Create `internal/repository/repository_test.go`:
```go
//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *PostgresRepository {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/analytics?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	repo := NewPostgresRepository(db)
	err = repo.RunMigrations(context.Background())
	require.NoError(t, err)

	// Clean up before each test
	_, err = db.Exec("TRUNCATE TABLE user_logins RESTART IDENTITY")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return repo
}

func TestInsertLogin_Success(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	loginTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	err := repo.InsertLogin(ctx, userID, loginTime)
	assert.NoError(t, err)
}

func TestInsertLogin_DuplicateIsIdempotent(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	loginTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	err := repo.InsertLogin(ctx, userID, loginTime)
	assert.NoError(t, err)

	// Insert same login again — should not error
	err = repo.InsertLogin(ctx, userID, loginTime)
	assert.NoError(t, err)
}

func TestGetDailyUniqueUsers_Basic(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Two different users login on Jan 15
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user2, time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC))

	// Same user logs in again on Jan 15 (different time)
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC))

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // 2 unique users, not 3 logins
}

func TestGetDailyUniqueUsers_NoLogins(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestGetDailyUniqueUsers_DayBoundary(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// user1 logs in at 23:59 on Jan 15
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC))
	// user2 logs in at 00:00 on Jan 16
	repo.InsertLogin(ctx, user2, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC))

	// Jan 15 should have 1 user
	date15 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date15, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Jan 16 should have 1 user
	date16 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	count, err = repo.GetDailyUniqueUsers(ctx, date16, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetDailyUniqueUsers_TimezoneAware(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()

	// Login at Jan 15 23:00 UTC = Jan 16 01:00 in Europe/Berlin (UTC+2 in summer, UTC+1 in winter)
	// January is winter, so UTC+1. 23:00 UTC = 00:00 CET Jan 16
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC))

	// In UTC, this is Jan 15
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	countUTC, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, countUTC)

	// In Europe/Berlin (CET, UTC+1 in Jan), 23:00 UTC = 00:00 Jan 16 CET
	// So Jan 15 in Berlin should have 0 users
	countBerlin15, err := repo.GetDailyUniqueUsers(ctx, date, "Europe/Berlin")
	assert.NoError(t, err)
	assert.Equal(t, 0, countBerlin15)

	// Jan 16 in Berlin should have 1 user
	date16 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	countBerlin16, err := repo.GetDailyUniqueUsers(ctx, date16, "Europe/Berlin")
	assert.NoError(t, err)
	assert.Equal(t, 1, countBerlin16)
}

func TestGetMonthlyUniqueUsers_Basic(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	// 3 users login across January
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user2, time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user3, time.Date(2024, 1, 31, 23, 0, 0, 0, time.UTC))
	// user1 logs in again in January
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 20, 8, 0, 0, 0, time.UTC))

	count, err := repo.GetMonthlyUniqueUsers(ctx, 2024, 1, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 3, count) // 3 unique users
}

func TestGetMonthlyUniqueUsers_MonthBoundary(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// user1 at Jan 31 23:59 UTC
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC))
	// user2 at Feb 1 00:00 UTC
	repo.InsertLogin(ctx, user2, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))

	// January should have 1 user
	janCount, err := repo.GetMonthlyUniqueUsers(ctx, 2024, 1, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, janCount)

	// February should have 1 user
	febCount, err := repo.GetMonthlyUniqueUsers(ctx, 2024, 2, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, febCount)
}

func TestDuplicateLoginSameUserDifferentTimes(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()

	// Same user, different timestamps on same day
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user1, time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC))

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Only 1 unique user despite 3 logins
}
```

- [ ] **Step 2: Run integration tests (requires running PostgreSQL)**

```bash
docker-compose up -d db
# Wait for PostgreSQL to be ready
go test ./internal/repository/ -v -count=1 -tags=integration
```

Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add .
git commit -m "feat: add repository integration tests"
```

---

### Task 11: README.md

**Files:**
- Create: `README.md`

**Interfaces:**
- Consumes: all other tasks' outputs
- Produces: comprehensive README for submission

- [ ] **Step 1: Create README.md**

Create `README.md`:
````markdown
# User Analytics Service

A backend service in Go that tracks user login events and provides daily/monthly unique user counts via REST API, backed by PostgreSQL.

## Quick Start

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)

### Run the Service

```bash
docker-compose up --build
```

The service will be available at `http://localhost:8080`.

### Run Without Docker

Prerequisites: Go 1.22+, PostgreSQL 16

```bash
# Set environment variables (or use defaults)
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=analytics
export SERVER_PORT=8080

# Run the service
go run ./cmd/server
```

## API Usage

### Record a Login

```bash
curl -X POST http://localhost:8080/api/logins \
  -H "Content-Type: application/json" \
  -d '{"user_id": "550e8400-e29b-41d4-a716-446655440000", "login_time": "2024-01-15T14:30:00Z"}'
```

**Response** (201 Created):
```json
{"message": "login recorded"}
```

### Get Daily Unique Users

```bash
curl "http://localhost:8080/api/analytics/daily?date=2024-01-15&tz=UTC"
```

**Response** (200 OK):
```json
{
  "date": "2024-01-15",
  "timezone": "UTC",
  "unique_users": 42
}
```

### Get Monthly Unique Users

```bash
curl "http://localhost:8080/api/analytics/monthly?month=2024-01&tz=UTC"
```

**Response** (200 OK):
```json
{
  "month": "2024-01",
  "timezone": "UTC",
  "unique_users": 150
}
```

### Health Check

```bash
curl http://localhost:8080/health
```

**Response** (200 OK):
```json
{"status": "ok"}
```

## Design Decisions

### Architecture
Classic layered architecture: **Handler → Service → Repository → PostgreSQL**. Each layer communicates via Go interfaces, enabling easy unit testing with mocks.

### Database
- **`TIMESTAMPTZ`** stores all timestamps in UTC internally; PostgreSQL handles timezone conversion at query time via `AT TIME ZONE`.
- **Unique constraint** on `(user_id, login_time)` prevents exact duplicate events. `ON CONFLICT DO NOTHING` makes inserts idempotent.
- **`COUNT(DISTINCT user_id)`** computes uniqueness at query time — simple, correct, and performant for expected volumes.
- Separate **B-tree index on `login_time`** accelerates range-based aggregation queries.

### Timezone Handling
- All timestamps are stored in UTC.
- Query endpoints accept an optional `tz` parameter (IANA timezone, e.g., `America/New_York`).
- If omitted, defaults to UTC.
- PostgreSQL's `AT TIME ZONE` handles conversion, ensuring a login at 23:00 UTC is correctly attributed to the next day in UTC+2 timezones.

### Edge Cases
- **Duplicate logins**: Same `(user_id, login_time)` is silently ignored (idempotent).
- **Day/month boundaries**: Range queries use `>=` start and `<` end (half-open interval) to prevent off-by-one errors.
- **No data**: Returns `unique_users: 0`, not an error.

## SQL Schema

```sql
CREATE TABLE user_logins (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL,
    login_time  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_user_logins_unique
    ON user_logins (user_id, login_time);

CREATE INDEX idx_user_logins_login_time
    ON user_logins (login_time);
```

## Testing

### Unit Tests (no database required)

```bash
go test ./internal/service/ ./internal/handler/ -v -count=1
```

### Integration Tests (requires PostgreSQL)

```bash
# Start PostgreSQL
docker-compose up -d db

# Run integration tests
go test ./internal/repository/ -v -count=1 -tags=integration
```

### All Tests

```bash
make test
```

## Project Structure

```
├── cmd/server/main.go           # Entry point, dependency wiring
├── internal/
│   ├── config/config.go         # Environment-based configuration
│   ├── handler/
│   │   ├── handler.go           # Gin HTTP handlers
│   │   └── handler_test.go      # Handler unit tests
│   ├── model/model.go           # Domain types
│   ├── repository/
│   │   ├── repository.go        # Repository interface + PostgreSQL impl
│   │   └── repository_test.go   # Integration tests
│   └── service/
│       ├── service.go           # Business logic
│       └── service_test.go      # Unit tests
├── migrations/                  # SQL migrations
├── Dockerfile                   # Multi-stage build
├── docker-compose.yml           # PostgreSQL + Go service
├── Makefile                     # Common commands
└── README.md
```
````

- [ ] **Step 2: Commit**

```bash
git add .
git commit -m "docs: add comprehensive README"
```

---

## Verification Plan

### Automated Tests

1. **Unit tests (no DB needed):**
   ```bash
   go test ./internal/service/ ./internal/handler/ -v -count=1
   ```

2. **Integration tests (requires PostgreSQL):**
   ```bash
   docker-compose up -d db
   go test ./internal/repository/ -v -count=1 -tags=integration
   ```

3. **Build verification:**
   ```bash
   go build ./cmd/server/
   go vet ./...
   ```

### Manual Verification

1. Run `docker-compose up --build`
2. Test all endpoints with curl:
   - POST a few login events
   - GET daily unique users
   - GET monthly unique users
   - GET health check
3. Verify duplicate login handling (POST same event twice)
4. Verify timezone query parameter works correctly
