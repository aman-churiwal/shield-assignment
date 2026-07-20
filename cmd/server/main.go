package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shield-assignment/internal/config"
	"github.com/shield-assignment/internal/handler"
	"github.com/shield-assignment/internal/repository"
	"github.com/shield-assignment/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := repository.ConnectDB(cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err.Error())
	}

	repo := repository.NewPostgresRepository(db)
	if err := repo.RunMigrations(context.Background()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations applied successfully.")

	svc := service.NewAnalyticsService(repo)
	h := handler.NewHandler(svc)

	router := gin.Default()
	h.RegisterRoutes(router)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown failed:", err)
	}

	if err := repo.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server stopped.")
}
