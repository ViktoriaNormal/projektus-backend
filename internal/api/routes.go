package api

import (
	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/api/middleware"
	"projektus-backend/internal/repositories"
	"projektus-backend/internal/services"
)

func SetupRouter(cfg *config.Config, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, notificationHandler *handlers.NotificationHandler, meetingHandler *handlers.MeetingHandler, roleHandler *handlers.RoleHandler, projectHandler *handlers.ProjectHandler, projectMemberHandler *handlers.ProjectMemberHandler, templateHandler *handlers.TemplateHandler, boardHandler *handlers.BoardHandler, taskHandler *handlers.TaskHandler, sprintHandler *handlers.SprintHandler, productBacklogHandler *handlers.ProductBacklogHandler, sprintBacklogHandler *handlers.SprintBacklogHandler, adminUserHandler *handlers.AdminUserHandler, adminPasswordPolicyHandler *handlers.AdminPasswordPolicyHandler, projectRoleHandler *handlers.ProjectRoleHandler, projectParamHandler *handlers.ProjectParamHandler, tagHandler *handlers.TagHandler, scrumAnalyticsHandler *handlers.ScrumAnalyticsHandler, kanbanAnalyticsHandler *handlers.KanbanAnalyticsHandler, projectService *services.ProjectService, permissionSvc *services.PermissionService) *gin.Engine {
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
			notifications.POST("/delete-all", notificationHandler.DeleteAll)
			notifications.POST("/:notificationId/read", notificationHandler.MarkAsRead)
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
			meetings.POST("/:meetingId/cancel", meetingHandler.CancelMeeting)

			meetings.GET("/:meetingId/participants", meetingHandler.ListParticipants)
			meetings.POST("/:meetingId/participants", meetingHandler.AddParticipants)
			meetings.POST("/:meetingId/response", meetingHandler.RespondToInvitation)
		}

		projects := v1.Group("/projects")
		projects.Use(middleware.AuthMiddleware(cfg))
		{
			projects.GET("/references", projectHandler.GetReferences)
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:projectId", projectHandler.GetProject)
			projects.PATCH("/:projectId", projectHandler.UpdateProject)
			projects.DELETE("/:projectId", projectHandler.DeleteProject)

			projects.GET("/:projectId/members", projectMemberHandler.ListMembers)
			projects.POST("/:projectId/members", projectMemberHandler.AddMember)
			projects.DELETE("/:projectId/members/:memberId", projectMemberHandler.RemoveMember)
			projects.PATCH("/:projectId/members/:memberId", projectMemberHandler.UpdateMemberRoles)

			// Project permissions
			projects.GET("/:projectId/my-permissions", projectRoleHandler.GetMyPermissions)

			// Project roles
			projects.GET("/:projectId/roles", projectRoleHandler.ListRoles)
			projects.POST("/:projectId/roles", projectRoleHandler.CreateRole)
			projects.PATCH("/:projectId/roles/:roleId", projectRoleHandler.UpdateRole)
			projects.DELETE("/:projectId/roles/:roleId", projectRoleHandler.DeleteRole)

			// Project params
			projects.GET("/:projectId/params", projectParamHandler.ListParams)
			projects.POST("/:projectId/params", projectParamHandler.CreateParam)
			projects.PATCH("/:projectId/params/:paramId", projectParamHandler.UpdateParam)
			projects.DELETE("/:projectId/params/:paramId", projectParamHandler.DeleteParam)

			projects.GET("/:projectId/sprints", sprintHandler.ListProjectSprints)
			projects.POST("/:projectId/sprints", sprintHandler.CreateSprint)

			projects.GET("/:projectId/backlog/product", productBacklogHandler.GetProductBacklog)
			projects.POST("/:projectId/backlog/product/tasks", productBacklogHandler.AddTaskToBacklog)
			projects.DELETE("/:projectId/backlog/product/tasks/:taskId", productBacklogHandler.RemoveTaskFromBacklog)
			projects.PATCH("/:projectId/backlog/product/reorder", productBacklogHandler.ReorderProductBacklog)

			projects.GET("/:projectId/backlog/sprint", sprintBacklogHandler.GetSprintBacklog)
			projects.POST("/:projectId/backlog/move-to-sprint", sprintBacklogHandler.MoveTasksToSprint)

			// Scrum analytics
			projects.GET("/:projectId/analytics/velocity", scrumAnalyticsHandler.GetVelocity)
			projects.GET("/:projectId/analytics/burndown", scrumAnalyticsHandler.GetBurndown)

			// Kanban analytics
			kanban := projects.Group("/:projectId/analytics/kanban")
			{
				kanban.GET("/summary", kanbanAnalyticsHandler.GetSummary)
				kanban.GET("/cumulative-flow", kanbanAnalyticsHandler.GetCumulativeFlow)
				kanban.GET("/cycle-time-scatter", kanbanAnalyticsHandler.GetCycleTimeScatter)
				kanban.GET("/throughput", kanbanAnalyticsHandler.GetThroughput)
				kanban.GET("/avg-cycle-time", kanbanAnalyticsHandler.GetAvgCycleTime)
				kanban.GET("/throughput-trend", kanbanAnalyticsHandler.GetThroughputTrend)
				kanban.GET("/wip", kanbanAnalyticsHandler.GetWipHistory)
				kanban.GET("/cycle-time-distribution", kanbanAnalyticsHandler.GetCycleTimeDistribution)
				kanban.GET("/throughput-distribution", kanbanAnalyticsHandler.GetThroughputDistribution)
				kanban.GET("/monte-carlo", kanbanAnalyticsHandler.GetMonteCarlo)
			}

		}

		sprints := v1.Group("/sprints")
		sprints.Use(middleware.AuthMiddleware(cfg))
		{
			sprints.GET("/:sprintId", sprintHandler.GetSprint)
			sprints.PATCH("/:sprintId", sprintHandler.UpdateSprint)
			sprints.DELETE("/:sprintId", sprintHandler.DeleteSprint)
			sprints.GET("/:sprintId/tasks", sprintHandler.GetSprintTasks)
			sprints.POST("/:sprintId/start", sprintHandler.StartSprint)
			sprints.POST("/:sprintId/complete", sprintHandler.CompleteSprint)
		}

		boards := v1.Group("/boards")
		boards.Use(middleware.AuthMiddleware(cfg))
		{
			boards.GET("", boardHandler.ListBoards)
			boards.POST("", boardHandler.CreateBoard)
			boards.PATCH("/reorder", boardHandler.ReorderBoards)
			boards.GET("/:boardId", boardHandler.GetBoard)
			boards.PATCH("/:boardId", boardHandler.UpdateBoard)
			boards.DELETE("/:boardId", boardHandler.DeleteBoard)

			// Columns
			boards.GET("/:boardId/columns", boardHandler.ListColumns)
			boards.POST("/:boardId/columns", boardHandler.CreateColumn)
			boards.PATCH("/:boardId/columns/reorder", boardHandler.ReorderColumns)
			boards.PATCH("/:boardId/columns/:columnId", boardHandler.UpdateColumn)
			boards.DELETE("/:boardId/columns/:columnId", boardHandler.DeleteColumn)

			// Swimlanes
			boards.GET("/:boardId/swimlanes", boardHandler.ListSwimlanes)
			boards.POST("/:boardId/swimlanes", boardHandler.CreateSwimlane)
			boards.PATCH("/:boardId/swimlanes/reorder", boardHandler.ReorderSwimlanes)
			boards.PATCH("/:boardId/swimlanes/:swimlaneId", boardHandler.UpdateSwimlane)
			boards.DELETE("/:boardId/swimlanes/:swimlaneId", boardHandler.DeleteSwimlane)

			// Notes
			boards.GET("/:boardId/notes", boardHandler.ListNotes)
			boards.POST("/columns/:columnId/notes", boardHandler.CreateNoteForColumn)
			boards.POST("/swimlanes/:swimlaneId/notes", boardHandler.CreateNoteForSwimlane)
			boards.PATCH("/notes/:noteId", boardHandler.UpdateNote)
			boards.DELETE("/notes/:noteId", boardHandler.DeleteNote)

			// Custom fields
			boards.GET("/:boardId/fields", boardHandler.ListCustomFields)
			boards.POST("/:boardId/fields", boardHandler.CreateCustomField)
			boards.PATCH("/:boardId/fields/:fieldId", boardHandler.UpdateCustomField)
			boards.DELETE("/:boardId/fields/:fieldId", boardHandler.DeleteCustomField)

			// Tags (board-scoped)
			boards.GET("/:boardId/tags", tagHandler.ListBoardTags)
			boards.POST("/:boardId/tasks/:taskId/tags", tagHandler.AddTagToTask)
			boards.PUT("/:boardId/tasks/:taskId/tags", tagHandler.SetTaskTags)
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
			tasks.DELETE("/:taskId/watchers/:memberId", taskHandler.RemoveWatcher)

			tasks.GET("/:taskId/dependencies", taskHandler.ListDependencies)
			tasks.POST("/:taskId/dependencies", taskHandler.AddDependency)

			tasks.GET("/:taskId/checklists", taskHandler.ListChecklists)
			tasks.POST("/:taskId/checklists", taskHandler.CreateChecklist)
			tasks.PATCH("/checklists/:checklistId", taskHandler.UpdateChecklist)
			tasks.DELETE("/checklists/:checklistId", taskHandler.DeleteChecklist)
			tasks.POST("/checklists/:checklistId/items", taskHandler.AddChecklistItem)
			tasks.PATCH("/checklist-items/:itemId", taskHandler.UpdateChecklistItem)
			tasks.PATCH("/checklist-items/:itemId/status", taskHandler.SetChecklistItemStatus)
			tasks.DELETE("/checklist-items/:itemId", taskHandler.DeleteChecklistItem)

			// Comments
			tasks.GET("/:taskId/comments", taskHandler.ListComments)
			tasks.POST("/:taskId/comments", taskHandler.CreateComment)
			tasks.DELETE("/comments/:commentId", taskHandler.DeleteComment)

			// Attachments
			tasks.GET("/:taskId/attachments", taskHandler.ListAttachments)
			tasks.POST("/:taskId/attachments", taskHandler.UploadAttachment)
			tasks.GET("/attachments/:attachmentId/download", taskHandler.DownloadAttachment)
			tasks.DELETE("/attachments/:attachmentId", taskHandler.DeleteAttachment)

			// Field values
			tasks.GET("/:taskId/field-values", taskHandler.GetTaskFieldValues)
			tasks.PUT("/:taskId/field-values/:fieldId", taskHandler.SetTaskFieldValue)

			// Tags (task-scoped)
			tasks.GET("/:taskId/tags", tagHandler.ListTaskTags)
			tasks.DELETE("/:taskId/tags/:tagId", tagHandler.RemoveTagFromTask)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg))
		{
			// Roles — require system.roles.manage
			roles := admin.Group("/roles")
			roles.Use(middleware.RequireSystemPermission(repositories.SystemPermissionManageRoles, permissionSvc))
			{
				roles.GET("", roleHandler.ListSystemRoles)
				roles.POST("", roleHandler.CreateSystemRole)
				roles.GET("/:roleId", roleHandler.GetRole)
				roles.PUT("/:roleId", roleHandler.UpdateSystemRole)
				roles.DELETE("/:roleId", roleHandler.DeleteRole)
			}

			// Users — require system.users.manage
			adminUsers := admin.Group("/users")
			adminUsers.Use(middleware.RequireSystemPermission(repositories.SystemPermissionManageUsers, permissionSvc))
			{
				adminUsers.GET("", adminUserHandler.ListUsers)
				adminUsers.POST("", adminUserHandler.CreateUser)
				adminUsers.GET("/:id", adminUserHandler.GetUser)
				adminUsers.PUT("/:id", adminUserHandler.UpdateUser)
				adminUsers.DELETE("/:id", adminUserHandler.DeleteUser)
			}

			// Password policy — require system.password_policy.manage
			passwordPolicy := admin.Group("/password-policy")
			passwordPolicy.Use(middleware.RequireSystemPermission(repositories.SystemPermissionManagePasswordPolicy, permissionSvc))
			{
				passwordPolicy.GET("", adminPasswordPolicyHandler.GetPasswordPolicy)
				passwordPolicy.PUT("", adminPasswordPolicyHandler.UpdatePasswordPolicy)
			}

			// Project templates — require system.project_templates.manage
			templates := admin.Group("/project-templates")
			templates.Use(middleware.RequireSystemPermission(repositories.SystemPermissionManageTemplates, permissionSvc))
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
				templates.POST("/:templateId/boards/:boardId/swimlanes", templateHandler.CreateSwimlane)
				templates.PATCH("/:templateId/boards/:boardId/swimlanes/reorder", templateHandler.ReorderSwimlanes)
				templates.PATCH("/:templateId/boards/:boardId/swimlanes/:swimlaneId", templateHandler.UpdateSwimlane)
				templates.DELETE("/:templateId/boards/:boardId/swimlanes/:swimlaneId", templateHandler.DeleteSwimlane)

				// Custom fields
				templates.POST("/:templateId/boards/:boardId/fields", templateHandler.CreateCustomField)
				templates.PATCH("/:templateId/boards/:boardId/fields/:fieldId", templateHandler.UpdateCustomField)
				templates.DELETE("/:templateId/boards/:boardId/fields/:fieldId", templateHandler.DeleteCustomField)

				// Project params
				templates.POST("/:templateId/project-params", templateHandler.CreateProjectParam)
				templates.PATCH("/:templateId/project-params/:paramId", templateHandler.UpdateProjectParam)
				templates.DELETE("/:templateId/project-params/:paramId", templateHandler.DeleteProjectParam)

				// Roles
				templates.POST("/:templateId/roles", templateHandler.CreateRole)
				templates.PATCH("/:templateId/roles/:roleId", templateHandler.UpdateRole)
				templates.DELETE("/:templateId/roles/:roleId", templateHandler.DeleteRole)
			}
		}
	}

	return r
}

