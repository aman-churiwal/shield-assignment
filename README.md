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

### Loading Sample Data

`migrations/002_sample_data.sql` contains a small fixture dataset that exercises
duplicate logins, day/month/year boundaries, and timezone-dependent bucketing.

```bash
Powershell Command:
Get-Content .\migrations\002_sample_data.sql -Raw | docker exec -i postgres psql -U postgres -d analytics

CMD command:
docker exec -i postgres psql -U postgres -d analytics < migrations\002_sample_data.sql
```

### Running Tests
# Unit tests only (service + handler layers, no DB required):
go test ./internal/service/... ./internal/handler/...

```bash
# Unit tests only (service + handler layers, no DB required):
go test ./internal/service/... ./internal/handler/...

# Full suite, including repository integration tests (needs a live Postgres: the docker-compose one):
docker-compose up -d postgres
go test ./...
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

## Design Decisions

### Assumptions
- A "login event" is submitted by the caller with an explicit `login_time`
  (no server-side time is recorded); the service does not itself
  authenticate the request or validate that `user_id` exists elsewhere. It assumes it is just an analytic service
- "Unique user" means unique `user_id`, not unique session/device. Repeat
    logins by the same user within the same day/month only count once.


### Architecture
**Handler → Service → Repository → PostgreSQL**

### Database
- **`TIMESTAMPTZ`** stores all timestamps in UTC internally; PostgreSQL handles timezone conversion at query time via `AT TIME ZONE`.
- **Unique constraint** on `(user_id, login_time)` prevents exact duplicate events. `ON CONFLICT DO NOTHING` makes inserts idempotent.
- **`COUNT(DISTINCT user_id)`** computes uniqueness at query time — simple, correct, and performant for expected volumes.
- Separate **index on `login_time`** speeds up the range-based aggregation queries.

### Timezone Handling
- All timestamps are stored in UTC.
- Query endpoint accepts an optional `tz` parameter (IANA timezone, e.g., `Asia/Kolkata`).
- If `tz` is not provided, then it defaults to UTC.
- **Date** is passed in the query string as `date` or `month`.

### Supporting Aggregation Queries
- `COUNT(DISTINCT user_id)` over the indexed `login_time` range is simple and fast at small volumes of data (an index range scan followed by a distinct-count over that range).
- Once the table grows to hundreds of millions of rows or if daily/monthly counts are requested very frequently (e.g., every few seconds), normal scanning over indexed `login_time` will not be efficient.
- To scale this, we can have a job which aggregates count for daily unique users and put them in `daily_unique_users(date, timezone, unique_users)` table.
- Similarly, we can have a job which aggregates count for monthly unique users and put them in `monthly_unique_users(month, timezone, unique_users)` table.
- Tradeoffs for the above two points are: first counts may lag behind real time logins. Secondly, late-arriving events require updating previous days or months.

### Edge Cases
- **Duplicate Logins**: Same `(user_id, login_time)` is idempotent.
- **Day/month boundaries**: Range queries use `>=` start and `<` end (half-open interval) to prevent off-by-one errors.
- **No data**: Returns `unique_users: 0`, not an error.


## SQL Schema
```sql
CREATE TABLE IF NOT EXISTS user_logins (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    login_time timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_logins_unique
    ON user_logins (user_id, login_time);

CREATE INDEX IF NOT EXISTS idx_user_logins_login_time
    ON user_logins (login_time);
```

## Project Structure
```
├── cmd/server/main.go              # Entry point, dependency wiring
├── internal/
│   ├── config/config.go            # Environment-based configuration
│   ├── handler/
│   │   ├── handler.go              # Gin HTTP handlers
│   │   └── handler_test.go         # Handler unit tests
│   ├── model/model.go              # Domain types
│   ├── repository/
│   │   ├── repository.go           # Repository interface + PostgreSQL impl
│   │   └── repository_test.go      # Integration tests
│   └── service/
│       ├── service.go              # Business logic
│       └── service_test.go         # Unit tests
├── migrations/                     # SQL migrations
│   ├── 001_create_user_logins.sql  # Sample Schema creation query
│   └── 002_sample_data.sql         # Sample Data Insertion query
├── Dockerfile                      # Multi-stage build
├── docker-compose.yml              # PostgreSQL + Go service
└── README.md
```