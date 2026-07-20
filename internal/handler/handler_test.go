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
	"github.com/google/uuid"
	"github.com/shield-assignment/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) RecordLogin(ctx context.Context, userID uuid.UUID, loginTime string) error {
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

func TestRecordLogin_Success(t *testing.T) {
	mockSvc := new(MockService)
	r := setupRouter(mockSvc)
	body := `{"user_id":"550e8400-e29b-41d4-a716-446655440000","login_time":"2024-01-15T14:30:00Z"}`
	mockSvc.On("RecordLogin", mock.Anything, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), "2024-01-15T14:30:00Z").Return(nil)
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
	body := `{"user_id":"660e8400-e29b-41d4-a716-446655440001","login_time":"2024-01-15T14:30:00Z"}`
	mockSvc.On("RecordLogin", mock.Anything, uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"), "2024-01-15T14:30:00Z").Return(errors.New("invalid user_id"))
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
