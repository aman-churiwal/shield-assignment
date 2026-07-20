package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func (m *MockRepository) GetMonthlyUniqueUsers(ctx context.Context, year, month int, timezone string) (int, error) {
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

	userID := uuid.New()
	loginTime := "2026-07-20T20:12:00Z"

	parsedTime, _ := time.Parse(time.RFC3339, loginTime)

	mockRepo.On("InsertLogin", mock.Anything, userID, parsedTime.UTC()).Return(nil)

	err := svc.RecordLogin(context.Background(), userID, "2026-07-20T20:12:00Z")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRecordLogin_InvalidTime(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	err := svc.RecordLogin(context.Background(), uuid.New(), "not-a-time")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid login_time")
}

func TestGetDailyUniqueUsers_ValidInput(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	expectedDate, _ := time.Parse("2006-01-02", "2026-07-20")
	mockRepo.On("GetDailyUniqueUsers", mock.Anything, expectedDate, "UTC").Return(42, nil)

	count, err := svc.GetDailyUniqueUsers(context.Background(), "2026-07-20", "UTC")
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

	expectedDate, _ := time.Parse("2006-01-02", "2026-07-20")
	mockRepo.On("GetDailyUniqueUsers", mock.Anything, expectedDate, "UTC").Return(10, nil)

	count, err := svc.GetMonthlyUniqueUsers(context.Background(), "2026-07-20", "")
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestGetMonthlyUniqueUsers_ValidInput(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewAnalyticsService(mockRepo)

	mockRepo.On("GetDailyUniqueUsers", mock.Anything, 2026, 1, "UTC").Return(150, nil)

	count, err := svc.GetMonthlyUniqueUsers(context.Background(), "2026-07-20", "UTC")
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

	expectedDate, _ := time.Parse("2006-01-02", "2026-07-20")
	mockRepo.On("GetDailyUniqueUsers", mock.Anything, expectedDate, "UTC").Return(0, fmt.Errorf("db error"))

	_, err := svc.GetDailyUniqueUsers(context.Background(), "2026-07-20", "UTC")
	assert.Error(t, err)
}
