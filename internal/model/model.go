package model

import (
	"time"

	"github.com/google/uuid"
)

type LoginEvent struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	LoginTime time.Time `json:"login_time"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	UserID    uuid.UUID `json:"user_id" binding:"required"`
	LoginTime string    `json:"login_time" binding:"required"`
}

type DailyUniqueUsersResponse struct {
	Date        string `json:"date"`
	TimeZone    string `json:"timezone"`
	UniqueUsers int    `json:"unique_users"`
}

type MonthlyUniqueUsersResponse struct {
	Date        string `json:"date"`
	TimeZone    string `json:"timezone"`
	UniqueUsers int    `json:"unique_users"`
}
