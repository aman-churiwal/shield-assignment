# User Analytics Service — Design Spec

## Overview

A backend service in Go that tracks user login events and provides unique user counts per day and per month. Uses PostgreSQL for persistent storage, exposes a RESTful HTTP API, and is packaged with Docker Compose for easy local and cloud deployment.

## Goals

- Efficient ingestion of user login events via REST API
- Accurate unique user counts per day and per month
- Correct handling of duplicate logins, day/month boundaries, and timezones
- Clean, modular, production-ready Go code with comprehensive unit tests
- Docker Compose setup for one-command local deployment

## Architecture

Classic layered architecture with clean separation of concerns:

```
Handler (Gin) → Service (Business Logic) → Repository (SQL/pgx) → PostgreSQL
```

- **Handlers**: Parse HTTP requests, validate input, format responses
- **Service**: Timezone conversion, business rules, orchestration
- **Repository**: Raw SQL queries via `database/sql` + `pgx` driver
- **Config**: Environment-based configuration (DB connection, port, etc.)

Each layer depends only on the layer below it via interfaces, enabling easy mocking and testing.

## API Endpoints

| Method | Path                     | Description                              |
|--------|--------------------------|------------------------------------------|
| POST   | `/api/logins`            | Ingest a user login event                |
| GET    | `/api/analytics/daily`   | Count unique users for a given date      |
| GET    | `/api/analytics/monthly` | Count unique users for a given month     |
| GET    | `/health`                | Health/readiness check                   |

### POST /api/logins

**Request body:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "login_time": "2024-01-15T14:30:00Z"
}
```

**Responses:**
- `201 Created` — login recorded
- `200 OK` — duplicate login (same user_id + login_time), idempotent
- `400 Bad Request` — invalid input (missing fields, bad UUID, bad timestamp)

### GET /api/analytics/daily

**Query parameters:**
- `date` (required): `YYYY-MM-DD` format, e.g., `2024-01-15`
- `tz` (optional): IANA timezone, e.g., `America/New_York`. Defaults to `UTC`

**Response:**
```json
{
  "date": "2024-01-15",
  "timezone": "UTC",
  "unique_users": 42
}
```

### GET /api/analytics/monthly

**Query parameters:**
- `month` (required): `YYYY-MM` format, e.g., `2024-01`
- `tz` (optional): IANA timezone. Defaults to `UTC`

**Response:**
```json
{
  "month": "2024-01",
  "timezone": "UTC",
  "unique_users": 150
}
```

### GET /health

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

## Database Schema

```sql
CREATE TABLE user_logins (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL,
    login_time  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent exact duplicate logins
CREATE UNIQUE INDEX idx_user_logins_unique
    ON user_logins (user_id, login_time);

-- Speed up time-range aggregation queries
CREATE INDEX idx_user_logins_login_time
    ON user_logins (login_time);
```

### Schema Decisions

- **`TIMESTAMPTZ`** over `TIMESTAMP WITHOUT TIME ZONE`: PostgreSQL stores TIMESTAMPTZ in UTC internally and can convert at query time using `AT TIME ZONE`, enabling correct timezone-aware queries without application-side conversion.
- **Unique constraint on `(user_id, login_time)`**: Prevents storing exact duplicate events. Different timestamps from the same user are allowed (they represent distinct login events).
- **Separate `login_time` index**: The daily/monthly queries filter by time range (`WHERE login_time >= X AND login_time < Y`). A B-tree index on `login_time` makes these range scans efficient.
- **`COUNT(DISTINCT user_id)`**: Uniqueness per day/month is computed at query time. For the expected data volumes in this assignment, this is performant and avoids the complexity of maintaining pre-aggregated tables.

### Query Patterns

**Daily unique users (with timezone):**
```sql
SELECT COUNT(DISTINCT user_id)
FROM user_logins
WHERE login_time >= $1::DATE AT TIME ZONE $2
  AND login_time < ($1::DATE + INTERVAL '1 day') AT TIME ZONE $2;
```

**Monthly unique users (with timezone):**
```sql
SELECT COUNT(DISTINCT user_id)
FROM user_logins
WHERE login_time >= DATE_TRUNC('month', $1::DATE) AT TIME ZONE $2
  AND login_time < (DATE_TRUNC('month', $1::DATE) + INTERVAL '1 month') AT TIME ZONE $2;
```

## Project Structure

```
shield-assignment/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, dependency wiring
├── internal/
│   ├── config/
│   │   └── config.go            # Environment-based configuration
│   ├── handler/
│   │   ├── handler.go           # Gin HTTP handlers
│   │   └── handler_test.go      # Handler unit tests (mock service)
│   ├── model/
│   │   └── model.go             # Domain types (LoginEvent, responses)
│   ├── repository/
│   │   ├── repository.go        # Repository interface + PostgreSQL impl
│   │   └── repository_test.go   # Integration tests (real DB)
│   └── service/
│       ├── service.go           # Business logic + service interface
│       └── service_test.go      # Unit tests (mock repository)
├── migrations/
│   └── 001_create_user_logins.sql
├── docker-compose.yml           # PostgreSQL + Go service
├── Dockerfile                   # Multi-stage build
├── go.mod
├── Makefile                     # Common commands
└── README.md                    # Setup, design decisions, usage examples
```

## Testing Strategy

### Unit Tests (Service Layer)
Mock the repository interface. Test:
- Correct timezone conversion in queries
- Proper date/month parsing
- Error propagation from repository

### Handler Tests
Mock the service interface. Test:
- Valid requests return correct status codes and response bodies
- Missing/invalid query params return 400
- Duplicate login returns 200 (idempotent)
- Valid new login returns 201

### Integration Tests (Repository Layer)
Use a real PostgreSQL instance (Docker). Test:
- Insert and query round-trip
- Duplicate login insert behavior (upsert/conflict handling)
- Daily count accuracy with multiple users and timestamps
- Monthly count accuracy across day boundaries
- Timezone-aware query correctness

### Edge Cases (Explicitly Tested)
- **Duplicate logins**: Same user_id + login_time → recorded once, no error
- **Day boundary**: Login at 23:59:59 UTC counted for that day, 00:00:00 UTC for next day
- **Month boundary**: Jan 31 23:59 vs Feb 1 00:00
- **Timezone effect**: Login at Jan 15 23:00 UTC is Jan 16 in UTC+2 — daily count should reflect the queried timezone
- **No logins**: Returns `unique_users: 0` (not an error)
- **Invalid input**: Bad UUID format, invalid date format, future dates (allow)

## Configuration

Environment variables:
- `DB_HOST` — PostgreSQL host (default: `localhost`)
- `DB_PORT` — PostgreSQL port (default: `5432`)
- `DB_USER` — Database user (default: `postgres`)
- `DB_PASSWORD` — Database password (default: `postgres`)
- `DB_NAME` — Database name (default: `analytics`)
- `SERVER_PORT` — HTTP server port (default: `8080`)

## Docker Setup

- **docker-compose.yml**: Two services — `db` (PostgreSQL 16) and `app` (Go service)
- **Dockerfile**: Multi-stage build (Go builder → minimal `alpine` runtime)
- Database schema applied automatically on startup via migration in `main.go`
- Health check endpoint for container orchestration readiness

## Non-Goals

- Authentication/authorization
- Rate limiting
- Pagination for login history
- Real-time analytics / WebSocket
- Pre-aggregated summary tables (CQRS pattern)
