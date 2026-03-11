package main

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"projektus-backend/config"
	"projektus-backend/internal/api"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/db"
	"projektus-backend/internal/repositories"
	"projektus-backend/internal/services"
)

func main() {
	cfg := config.Load()

	conn, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer conn.Close()

	if err := conn.PingContext(context.Background()); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	queries := db.New(conn)

	userRepo := repositories.NewUserRepository(queries)
	authRepo := repositories.NewAuthRepository(queries)

	passwordSvc := services.NewPasswordService()
	rateLimitSvc := services.NewRateLimitService(cfg, authRepo)
	authSvc := services.NewAuthService(cfg, userRepo, authRepo, passwordSvc, rateLimitSvc)

	authHandler := handlers.NewAuthHandler(authSvc)

	router := api.SetupRouter(cfg, authHandler)

	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
