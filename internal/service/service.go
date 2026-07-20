package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shield-assignment/internal/repository"
)

type AnalyticsService interface {
	RecordLogin(ctx context.Context, userID uuid.UUID, loginTime string) error
	GetDailyUniqueUsers(ctx context.Context, date, timezone string) (int, error)
	GetMonthlyUniqueUsers(ctx context.Context, month, timezone string) (int, error)
}

type analyticsService struct {
	repo repository.Repository
}

func NewAnalyticsService(repo repository.Repository) AnalyticsService {
	return &analyticsService{repo: repo}
}

func (s *analyticsService) RecordLogin(ctx context.Context, userID uuid.UUID, loginTime string) error {
	parsedTime, err := time.Parse(time.RFC3339, loginTime)
	if err != nil {
		return fmt.Errorf("invalid login_time: %w", err)
	}

	return s.repo.InsertLogin(ctx, userID, parsedTime.UTC())
}

func (s *analyticsService) GetDailyUniqueUsers(ctx context.Context, date, timezone string) (int, error) {
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

func (s *analyticsService) GetMonthlyUniqueUsers(ctx context.Context, month, timezone string) (int, error) {
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
