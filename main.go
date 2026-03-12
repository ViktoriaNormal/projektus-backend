package main

import (
	"context"
	"database/sql"
	"log"
	"time"

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
	notificationRepo := repositories.NewNotificationRepository(queries)
	meetingRepo := repositories.NewMeetingRepository(queries)

	passwordSvc := services.NewPasswordService()
	rateLimitSvc := services.NewRateLimitService(cfg, authRepo)
	authSvc := services.NewAuthService(cfg, userRepo, authRepo, passwordSvc, rateLimitSvc)

	authHandler := handlers.NewAuthHandler(authSvc)
	userSvc := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)
	notificationSvc := services.NewNotificationService(notificationRepo)
	meetingSvc := services.NewMeetingService(meetingRepo, notificationSvc)
	meetingHandler := handlers.NewMeetingHandler(meetingSvc)

	router := api.SetupRouter(cfg, authHandler, userHandler, meetingHandler)

	// Фоновый воркер для напоминаний о встречах.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for now := range ticker.C {
			ctx := context.Background()

			userIDs, err := userRepo.ListAllUserIDs(ctx)
			if err != nil {
				log.Printf("failed to list user ids for reminders: %v", err)
				continue
			}

			for _, uid := range userIDs {
				if err := meetingSvc.CheckAndSendMeetingRemindersForUser(ctx, uid, now.UTC(), 5*time.Minute); err != nil {
					log.Printf("failed to send meeting reminders for user %s: %v", uid, err)
				}
			}
		}
	}()

	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
