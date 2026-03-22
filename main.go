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
	roleRepo := repositories.NewRoleRepository(queries)
	projectRepo := repositories.NewProjectRepository(queries)
	projectMemberRepo := repositories.NewProjectMemberRepository(queries)
	templateRepo := repositories.NewTemplateRepository(queries)
	boardRepo := repositories.NewBoardRepository(queries)
	taskRepo := repositories.NewTaskRepository(queries)
	commentRepo := repositories.NewCommentRepository(queries)
	attachmentRepo := repositories.NewAttachmentRepository(queries)
	sprintRepo := repositories.NewSprintRepository(queries)
	productBacklogRepo := repositories.NewProductBacklogRepository(queries)
	sprintTaskRepo := repositories.NewSprintTaskRepository(queries)
	classOfServiceRepo := repositories.NewClassOfServiceRepository(queries)
	swimlaneConfigRepo := repositories.NewSwimlaneConfigRepository(queries)
	kanbanRepo := repositories.NewKanbanRepository(queries)
	taskHistoryRepo := repositories.NewTaskHistoryRepository(queries)
	forecastRepo := repositories.NewForecastRepository(queries)
	analyticsCacheRepo := repositories.NewAnalyticsCacheRepository(queries)
	scrumAnalyticsRepo := repositories.NewScrumAnalyticsRepository(queries)
	kanbanAnalyticsRepo := repositories.NewKanbanAnalyticsRepository(queries)
	adminUserRepo := repositories.NewAdminUserRepository(queries)
	passwordPolicyRepo := repositories.NewPasswordPolicyRepository(queries)
	roleSvc := services.NewRoleService(roleRepo)
	permissionSvc := services.NewPermissionService(roleSvc)
	passwordSvc := services.NewPasswordService()
	passwordPolicySvc := services.NewPasswordPolicyService(passwordPolicyRepo)
	rateLimitSvc := services.NewRateLimitService(cfg, authRepo)
	authSvc := services.NewAuthService(cfg, userRepo, authRepo, passwordSvc, passwordPolicySvc, rateLimitSvc, roleSvc)

	authHandler := handlers.NewAuthHandler(authSvc, roleSvc)
	userSvc := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc, projectMemberRepo, roleRepo)
	notificationSvc := services.NewNotificationService(notificationRepo)
	notificationHandler := handlers.NewNotificationHandler(notificationSvc)
	meetingSvc := services.NewMeetingService(meetingRepo, notificationSvc)
	meetingHandler := handlers.NewMeetingHandler(meetingSvc)

	scrumRoleSvc := services.NewScrumRoleService(roleRepo)

	roleHandler := handlers.NewRoleHandler(roleSvc)
	projectSvc := services.NewProjectService(projectRepo, scrumRoleSvc)
	projectHandler := handlers.NewProjectHandler(projectSvc)

	boardSvc := services.NewBoardService(boardRepo)
	boardHandler := handlers.NewBoardHandler(boardSvc, projectSvc)

	taskHistorySvc := services.NewTaskHistoryService(taskHistoryRepo)

	taskSvc := services.NewTaskService(taskRepo, projectRepo, taskHistorySvc)
	taskHandler := handlers.NewTaskHandler(taskSvc)

	classOfServiceSvc := services.NewClassOfServiceService(classOfServiceRepo, swimlaneConfigRepo, boardRepo)
	classOfServiceHandler := handlers.NewClassOfServiceHandler(classOfServiceSvc)

	kanbanSvc := services.NewKanbanService(kanbanRepo, boardRepo)
	kanbanHandler := handlers.NewKanbanHandler(kanbanSvc)

	forecastSvc := services.NewMonteCarloForecastService(taskHistoryRepo, forecastRepo)
	forecastHandler := handlers.NewForecastHandler(forecastSvc)

	scrumAnalyticsSvc := services.NewScrumAnalyticsService(scrumAnalyticsRepo, sprintRepo, analyticsCacheRepo)
	scrumAnalyticsHandler := handlers.NewScrumAnalyticsHandler(scrumAnalyticsSvc)

	kanbanAnalyticsSvc := services.NewKanbanAnalyticsService(kanbanAnalyticsRepo, boardRepo, analyticsCacheRepo)
	kanbanAnalyticsHandler := handlers.NewKanbanAnalyticsHandler(kanbanAnalyticsSvc)

	adminUserSvc := services.NewAdminUserService(userRepo, adminUserRepo, roleSvc, passwordSvc, passwordPolicySvc)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserSvc)
	adminPasswordPolicyHandler := handlers.NewAdminPasswordPolicyHandler(passwordPolicySvc)

	commentSvc := services.NewCommentService(commentRepo, projectMemberRepo)
	commentHandler := handlers.NewCommentHandler(commentSvc)

	attachmentSvc := services.NewAttachmentService(attachmentRepo)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentSvc)

	sprintSvc := services.NewSprintService(sprintRepo, sprintTaskRepo, productBacklogRepo, taskRepo)
	sprintHandler := handlers.NewSprintHandler(sprintSvc)

	productBacklogSvc := services.NewProductBacklogService(productBacklogRepo, taskRepo)
	productBacklogHandler := handlers.NewProductBacklogHandler(productBacklogSvc)
	sprintBacklogHandler := handlers.NewSprintBacklogHandler(sprintSvc)

	projectMemberSvc := services.NewProjectMemberService(projectMemberRepo, userRepo, roleRepo)
	projectMemberHandler := handlers.NewProjectMemberHandler(projectMemberSvc)

	referenceRepo := repositories.NewReferenceRepository(queries)
	templateSvc := services.NewTemplateService(templateRepo, referenceRepo)
	templateHandler := handlers.NewTemplateHandler(templateSvc)

	router := api.SetupRouter(cfg, authHandler, userHandler, notificationHandler, meetingHandler, roleHandler, projectHandler, projectMemberHandler, templateHandler, boardHandler, taskHandler, commentHandler, attachmentHandler, sprintHandler, productBacklogHandler, sprintBacklogHandler, classOfServiceHandler, kanbanHandler, forecastHandler, scrumAnalyticsHandler, kanbanAnalyticsHandler, adminUserHandler, adminPasswordPolicyHandler, projectSvc, permissionSvc)

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
