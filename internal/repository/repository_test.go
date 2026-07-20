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

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/analytics?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	repo := NewPostgresRepository(db)
	err = repo.RunMigrations(context.Background())
	require.NoError(t, err)

	_, err = db.Exec("TRUNCATE TABLE user_logins RESTART IDENTITY")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return repo
}

func TestInsertLogin_Successful(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	loginTime := time.Date(2026, 7, 20, 21, 57, 0, 0, time.UTC)

	err := repo.InsertLogin(ctx, userID, loginTime)
	assert.NoError(t, err)
}

func TestInsertLogin_DuplicateIsIdempotent(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	loginTime := time.Date(2026, 7, 20, 21, 57, 0, 0, time.UTC)

	err := repo.InsertLogin(ctx, userID, loginTime)
	assert.NoError(t, err)

	err = repo.InsertLogin(ctx, userID, loginTime)
	assert.NoError(t, err)
}

func TestGetDailyUniqueUsers_Basic(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 21, 57, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user2, time.Date(2026, 7, 20, 22, 57, 0, 0, time.UTC))

	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 23, 57, 0, 0, time.UTC))

	date := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestGetDailyUniqueUsers_NoLogins(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	date := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestGetDailyUniqueUsers_DayBoundary(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// One at 11:59 PM
	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 23, 59, 59, 0, time.UTC))
	// Another at 12:00 AM
	repo.InsertLogin(ctx, user2, time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC))

	date20 := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date20, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	date21 := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	count, err = repo.GetDailyUniqueUsers(ctx, date21, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetDailyUniqueUsers_TimezoneAware(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()

	// Login at July 20 23:00 UTC = July 21 01:00 in Europe/Berlin (UTC+2 in summer, UTC+1 in winter)
	// July is summer, so UTC+2. 23:00 UTC = 01:00 CET July 21
	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 23, 0, 0, 0, time.UTC))

	// In UTC, this is July 20
	date := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	countUTC, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, countUTC)

	// In Europe/Berlin (CET, UTC+2 in July), 23:00 UTC = 01:00 July 21 CET
	// So July 20 in Berlin should have 0 users
	countBerlin20, err := repo.GetDailyUniqueUsers(ctx, date, "Europe/Berlin")
	assert.NoError(t, err)
	assert.Equal(t, 0, countBerlin20)

	// July 21 in Berlin should have 1 user
	date21 := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	countBerlin21, err := repo.GetDailyUniqueUsers(ctx, date21, "Europe/Berlin")
	assert.NoError(t, err)
	assert.Equal(t, 1, countBerlin21)
}

func TestGetMonthlyUniqueUsers_Basic(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 1, 21, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user2, time.Date(2026, 7, 8, 21, 0, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user3, time.Date(2026, 7, 20, 21, 0, 0, 0, time.UTC))

	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 25, 21, 0, 0, 0, time.UTC))

	count, err := repo.GetMonthlyUniqueUsers(ctx, 2026, 7, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestGetMonthlyUniqueUsers_MonthBoundary(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// One at 31st July 11:59PM
	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC))
	// Another at 1st August 12:00AM
	repo.InsertLogin(ctx, user2, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))

	julyCount, err := repo.GetMonthlyUniqueUsers(ctx, 2026, 7, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, julyCount)

	augustCount, err := repo.GetMonthlyUniqueUsers(ctx, 2026, 8, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, augustCount)
}

func TestDuplicateLoginSameUserDifferentTimes(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user1 := uuid.New()

	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 21, 57, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 22, 17, 0, 0, time.UTC))
	repo.InsertLogin(ctx, user1, time.Date(2026, 7, 20, 22, 50, 0, 0, time.UTC))

	date := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	count, err := repo.GetDailyUniqueUsers(ctx, date, "UTC")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}
