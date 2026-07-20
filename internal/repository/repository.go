package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repository interface {
	InsertLogin(ctx context.Context, userID uuid.UUID, loginTime time.Time) error
	GetDailyUniqueUsers(ctx context.Context, date time.Time, timezone string) (int, error)
	GetMonthlyUniqueUsers(ctx context.Context, year, month int, timezone string) (int, error)
	RunMigrations(ctx context.Context) error
	Close() error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

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
		WHERE login_time >= $1::date::timestamp AT TIME ZONE $2
		AND login_time < ($1::date::timestamp + INTERVAL '1 day') AT TIME ZONE $2
	`
	dateStr := date.Format("2006-01-02")
	var count int
	err := r.db.QueryRowContext(ctx, query, dateStr, timezone).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily unique users: %w", err)
	}

	return count, nil
}

func (r *PostgresRepository) GetMonthlyUniqueUsers(ctx context.Context, year, month int, timezone string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM user_logins
		WHERE login_time >= DATE_TRUNC('month', $1::date::timestamp) AT TIME ZONE $2
		AND login_time < (DATE_TRUNC('month', $1::date::timestamp) + INTERVAL '1 month') AT TIME ZONE $2
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
		    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
		    user_id uuid NOT NULL,
		    login_time timestamptz NOT NULL,
		    created_at timestamptz NOT NULL DEFAULT NOW()
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
