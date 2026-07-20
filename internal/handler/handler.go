package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shield-assignment/internal/model"
	"github.com/shield-assignment/internal/service"
)

type Handler struct {
	svc service.AnalyticsService
}

func NewHandler(svc service.AnalyticsService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/logins", h.RecordLogin)
		api.GET("/analytics/daily", h.GetDailyUniqueUsers)
		api.GET("/analytics/monthly", h.GetMonthlyUniqueUsers)
	}
}

func (h *Handler) RecordLogin(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("invalid request body: " + err.Error())})
		return
	}

	if err := h.svc.RecordLogin(c.Request.Context(), req.UserID, req.LoginTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}

	c.JSON(http.StatusCreated, gin.H{"message": "login recorded"})
}

func (h *Handler) GetDailyUniqueUsers(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("missing required query parameter: date")})
		return
	}

	tz := c.Query("tz")

	count, err := h.svc.GetDailyUniqueUsers(c.Request.Context(), date, tz)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
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

func (h *Handler) GetMonthlyUniqueUsers(c *gin.Context) {
	month := c.Query("month")
	if month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("missing required query parameter: month")})
		return
	}

	tz := c.Query("tz")

	count, err := h.svc.GetMonthlyUniqueUsers(c.Request.Context(), month, tz)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
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
