package api

import (
	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/api/middleware"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

func SetupRouter(cfg *config.Config, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, notificationHandler *handlers.NotificationHandler, meetingHandler *handlers.MeetingHandler, roleHandler *handlers.RoleHandler, projectHandler *handlers.ProjectHandler, projectMemberHandler *handlers.ProjectMemberHandler, templateHandler *handlers.TemplateHandler, boardHandler *handlers.BoardHandler, taskHandler *handlers.TaskHandler, commentHandler *handlers.CommentHandler, attachmentHandler *handlers.AttachmentHandler, sprintHandler *handlers.SprintHandler, productBacklogHandler *handlers.ProductBacklogHandler, sprintBacklogHandler *handlers.SprintBacklogHandler, classOfServiceHandler *handlers.ClassOfServiceHandler, kanbanHandler *handlers.KanbanHandler, forecastHandler *handlers.ForecastHandler, scrumAnalyticsHandler *handlers.ScrumAnalyticsHandler, kanbanAnalyticsHandler *handlers.KanbanAnalyticsHandler, adminUserHandler *handlers.AdminUserHandler, adminPasswordPolicyHandler *handlers.AdminPasswordPolicyHandler, projectService *services.ProjectService, permissionSvc *services.PermissionService) *gin.Engine {
	r := gin.Default()

	// Раздача статических файлов (аватары, вложения)
	r.Static("/uploads", "./uploads")

	v1 := r.Group("/api/v1")
	v1.Use(middleware.CORSMiddleware(cfg))
	v1.Use(middleware.AuditFileLogger("audit.log"))
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/password-policy", adminPasswordPolicyHandler.GetPasswordPolicy)

			protected := auth.Group("")
			protected.Use(middleware.AuthMiddleware(cfg))
			{
				protected.POST("/change-password", authHandler.ChangePassword)
			}
		}

		permissions := v1.Group("/permissions")
		permissions.Use(middleware.AuthMiddleware(cfg))
		{
			permissions.GET("", roleHandler.ListPermissions)
		}

		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(cfg))
		{
			users.GET("", userHandler.SearchUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PATCH("/:id", userHandler.UpdateUser)
			users.PUT("/:id/avatar", userHandler.UpdateAvatar)
			users.GET("/:id/roles", roleHandler.GetMySystemRoles)
			users.GET("/:id/project-roles", userHandler.GetMyProjectRoles)
		}

		notifications := v1.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware(cfg))
		{
			notifications.GET("", notificationHandler.GetFeed)
			notifications.POST("/read-all", notificationHandler.MarkAllAsRead)
			notifications.PATCH("/:notificationId/read", notificationHandler.MarkAsRead)
			notifications.GET("/settings", notificationHandler.GetSettings)
			notifications.PUT("/settings", notificationHandler.UpdateSettings)
		}

		meetings := v1.Group("/meetings")
		meetings.Use(middleware.AuthMiddleware(cfg))
		{
			meetings.GET("", meetingHandler.ListUserMeetings)
			meetings.POST("", meetingHandler.CreateMeeting)
			meetings.GET("/:meetingId", meetingHandler.GetMeeting)
			meetings.PATCH("/:meetingId", meetingHandler.UpdateMeeting)
			meetings.DELETE("/:meetingId", meetingHandler.CancelMeeting)

			meetings.GET("/:meetingId/participants", meetingHandler.ListParticipants)
			meetings.POST("/:meetingId/participants", meetingHandler.AddParticipants)
			meetings.POST("/:meetingId/response", meetingHandler.RespondToInvitation)
		}

		projects := v1.Group("/projects")
		projects.Use(middleware.AuthMiddleware(cfg))
		{
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:projectId", projectHandler.GetProject)
			projects.PATCH("/:projectId", projectHandler.UpdateProject)
			projects.DELETE("/:projectId", projectHandler.DeleteProject)

			projects.GET("/:projectId/members", projectMemberHandler.ListMembers)
			projects.POST("/:projectId/members", projectMemberHandler.AddMember)
			projects.DELETE("/:projectId/members/:memberId", projectMemberHandler.RemoveMember)
			projects.PATCH("/:projectId/members/:memberId", projectMemberHandler.UpdateMemberRoles)

			projects.GET("/:projectId/sprints", sprintHandler.ListProjectSprints)
			projects.POST("/:projectId/sprints", sprintHandler.CreateSprint)

			projects.GET("/:projectId/backlog/product", productBacklogHandler.GetProductBacklog)
			projects.POST("/:projectId/backlog/product/tasks", productBacklogHandler.AddTaskToBacklog)
			projects.DELETE("/:projectId/backlog/product/tasks/:taskId", productBacklogHandler.RemoveTaskFromBacklog)
			projects.PATCH("/:projectId/backlog/product/reorder", productBacklogHandler.ReorderProductBacklog)

			projects.GET("/:projectId/backlog/sprint", sprintBacklogHandler.GetSprintBacklog)
			projects.POST("/:projectId/backlog/move-to-sprint", sprintBacklogHandler.MoveTasksToSprint)

			projects.GET("/:projectId/classes-of-service", classOfServiceHandler.GetClassesOfService)

			projects.GET("/:projectId/kanban/wip-limits", kanbanHandler.GetWipLimits)
			projects.PUT("/:projectId/kanban/wip-limits", kanbanHandler.UpdateWipLimits)

			projects.POST("/:projectId/kanban/forecast", forecastHandler.GenerateForecast)

			scrumAnalytics := projects.Group("/:projectId/analytics/scrum")
			scrumAnalytics.Use(middleware.RequireProjectType(domain.ProjectTypeScrum, projectService))
			{
				scrumAnalytics.GET("/velocity", scrumAnalyticsHandler.GetVelocity)
				scrumAnalytics.GET("/burndown", scrumAnalyticsHandler.GetBurndown)
			}

			kanbanAnalytics := projects.Group("/:projectId/analytics/kanban")
			kanbanAnalytics.Use(middleware.RequireProjectType(domain.ProjectTypeKanban, projectService))
			{
				kanbanAnalytics.GET("/cumulative-flow", kanbanAnalyticsHandler.GetCumulativeFlow)
				kanbanAnalytics.GET("/throughput", kanbanAnalyticsHandler.GetThroughput)
				kanbanAnalytics.GET("/wip/over-time", kanbanAnalyticsHandler.GetWipOverTime)
				kanbanAnalytics.GET("/wip/age", kanbanAnalyticsHandler.GetWipAge)
				kanbanAnalytics.GET("/cycle-time/scatterplot", kanbanAnalyticsHandler.GetCycleTimeScatterplot)
				kanbanAnalytics.GET("/cycle-time/trend", kanbanAnalyticsHandler.GetCycleTimeTrend)
				kanbanAnalytics.GET("/cycle-time/histogram", kanbanAnalyticsHandler.GetCycleTimeHistogram)
				kanbanAnalytics.GET("/throughput/histogram", kanbanAnalyticsHandler.GetThroughputHistogram)
			}
		}

		sprints := v1.Group("/sprints")
		sprints.Use(middleware.AuthMiddleware(cfg))
		{
			sprints.GET("/:sprintId", sprintHandler.GetSprint)
			sprints.PATCH("/:sprintId", sprintHandler.UpdateSprint)
			sprints.DELETE("/:sprintId", sprintHandler.DeleteSprint)
			sprints.POST("/:sprintId/start", sprintHandler.StartSprint)
			sprints.POST("/:sprintId/complete", sprintHandler.CompleteSprint)
		}

		boards := v1.Group("/boards")
		boards.Use(middleware.AuthMiddleware(cfg))
		{
			boards.GET("", boardHandler.ListBoards)
			boards.POST("", boardHandler.CreateBoard)
			boards.GET("/:boardId", boardHandler.GetBoard)
			boards.PATCH("/:boardId", boardHandler.UpdateBoard)
			boards.DELETE("/:boardId", boardHandler.DeleteBoard)

			boards.GET("/:boardId/columns", boardHandler.ListColumns)
			boards.POST("/:boardId/columns", boardHandler.CreateColumn)

			boards.GET("/:boardId/swimlanes", boardHandler.ListSwimlanes)
			boards.POST("/:boardId/swimlanes", boardHandler.CreateSwimlane)

			boards.GET("/:boardId/notes", boardHandler.ListNotes)

			boards.POST("/columns/:columnId/notes", boardHandler.CreateNoteForColumn)
			boards.POST("/swimlanes/:swimlaneId/notes", boardHandler.CreateNoteForSwimlane)

			boards.POST("/:boardId/swimlanes/configure", classOfServiceHandler.ConfigureSwimlanes)

			boards.GET("/:boardId/wip-counts", kanbanHandler.GetCurrentWipCounts)
		}

		tasks := v1.Group("/tasks")
		tasks.Use(middleware.AuthMiddleware(cfg))
		{
			tasks.GET("", taskHandler.SearchTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.GET("/:taskId", taskHandler.GetTask)
			tasks.PATCH("/:taskId", taskHandler.UpdateTask)
			tasks.DELETE("/:taskId", taskHandler.DeleteTask)

			tasks.GET("/:taskId/watchers", taskHandler.ListWatchers)
			tasks.POST("/:taskId/watchers", taskHandler.AddWatcher)

			tasks.GET("/:taskId/dependencies", taskHandler.ListDependencies)
			tasks.POST("/:taskId/dependencies", taskHandler.AddDependency)

			tasks.GET("/:taskId/checklists", taskHandler.ListChecklists)
			tasks.POST("/:taskId/checklists", taskHandler.CreateChecklist)
			tasks.POST("/checklists/:checklistId/items", taskHandler.AddChecklistItem)
			tasks.PATCH("/checklist-items/:itemId/status", taskHandler.SetChecklistItemStatus)

			tasks.GET("/:taskId/comments", commentHandler.ListTaskComments)
			tasks.POST("/comments", commentHandler.CreateComment)

			tasks.GET("/:taskId/attachments", attachmentHandler.ListTaskAttachments)
			tasks.POST("/:taskId/attachments", attachmentHandler.UploadTaskAttachment)

			tasks.PATCH("/:taskId/class-of-service", classOfServiceHandler.UpdateTaskClass)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg))
		{
			// Roles — require system.roles.manage
			roles := admin.Group("/roles")
			roles.Use(middleware.RequireSystemPermission(services.SystemPermissionManageRoles, permissionSvc))
			{
				roles.GET("", roleHandler.ListSystemRoles)
				roles.POST("", roleHandler.CreateSystemRole)
				roles.GET("/:roleId", roleHandler.GetRole)
				roles.PUT("/:roleId", roleHandler.UpdateSystemRole)
				roles.DELETE("/:roleId", roleHandler.DeleteRole)
			}

			// Users — require system.users.manage
			adminUsers := admin.Group("/users")
			adminUsers.Use(middleware.RequireSystemPermission(services.SystemPermissionManageUsers, permissionSvc))
			{
				adminUsers.GET("", adminUserHandler.ListUsers)
				adminUsers.POST("", adminUserHandler.CreateUser)
				adminUsers.GET("/:id", adminUserHandler.GetUser)
				adminUsers.PUT("/:id", adminUserHandler.UpdateUser)
				adminUsers.DELETE("/:id", adminUserHandler.DeleteUser)
			}

			// Password policy — require system.password_policy.manage
			passwordPolicy := admin.Group("/password-policy")
			passwordPolicy.Use(middleware.RequireSystemPermission(services.SystemPermissionManagePasswordPolicy, permissionSvc))
			{
				passwordPolicy.GET("", adminPasswordPolicyHandler.GetPasswordPolicy)
				passwordPolicy.PUT("", adminPasswordPolicyHandler.UpdatePasswordPolicy)
			}

			// Project templates — require system.project_templates.manage
			templates := admin.Group("/project-templates")
			templates.Use(middleware.RequireSystemPermission(services.SystemPermissionManageTemplates, permissionSvc))
			{
				templates.GET("/references", templateHandler.GetReferences)
				templates.GET("", templateHandler.ListTemplates)
				templates.POST("", templateHandler.CreateTemplate)
				templates.GET("/:templateId", templateHandler.GetTemplate)
				templates.PATCH("/:templateId", templateHandler.UpdateTemplate)
				templates.DELETE("/:templateId", templateHandler.DeleteTemplate)

				// Boards
				templates.POST("/:templateId/boards", templateHandler.CreateBoard)
				templates.PATCH("/:templateId/boards/reorder", templateHandler.ReorderBoards)
				templates.PATCH("/:templateId/boards/:boardId", templateHandler.UpdateBoard)
				templates.DELETE("/:templateId/boards/:boardId", templateHandler.DeleteBoard)

				// Columns
				templates.POST("/:templateId/boards/:boardId/columns", templateHandler.CreateColumn)
				templates.PATCH("/:templateId/boards/:boardId/columns/reorder", templateHandler.ReorderColumns)
				templates.PATCH("/:templateId/boards/:boardId/columns/:columnId", templateHandler.UpdateColumn)
				templates.DELETE("/:templateId/boards/:boardId/columns/:columnId", templateHandler.DeleteColumn)

				// Swimlanes
				templates.PATCH("/:templateId/boards/:boardId/swimlanes/reorder", templateHandler.ReorderSwimlanes)
				templates.PATCH("/:templateId/boards/:boardId/swimlanes/:swimlaneId", templateHandler.UpdateSwimlane)
				templates.DELETE("/:templateId/boards/:boardId/swimlanes/:swimlaneId", templateHandler.DeleteSwimlane)

				// Priority values
				templates.PUT("/:templateId/boards/:boardId/priority-values", templateHandler.ReplacePriorityValues)

				// Custom fields
				templates.POST("/:templateId/boards/:boardId/custom-fields", templateHandler.CreateCustomField)
				templates.PATCH("/:templateId/boards/:boardId/custom-fields/reorder", templateHandler.ReorderCustomFields)
				templates.PATCH("/:templateId/boards/:boardId/custom-fields/:fieldId", templateHandler.UpdateCustomField)
				templates.DELETE("/:templateId/boards/:boardId/custom-fields/:fieldId", templateHandler.DeleteCustomField)
			}
		}
	}

	return r
}

