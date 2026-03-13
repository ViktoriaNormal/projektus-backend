package api

import (
	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/api/middleware"
	"projektus-backend/internal/services"
)

func SetupRouter(cfg *config.Config, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, meetingHandler *handlers.MeetingHandler, roleHandler *handlers.RoleHandler, projectHandler *handlers.ProjectHandler, projectMemberHandler *handlers.ProjectMemberHandler, templateHandler *handlers.TemplateHandler, boardHandler *handlers.BoardHandler, taskHandler *handlers.TaskHandler, commentHandler *handlers.CommentHandler, attachmentHandler *handlers.AttachmentHandler, sprintHandler *handlers.SprintHandler, productBacklogHandler *handlers.ProductBacklogHandler, sprintBacklogHandler *handlers.SprintBacklogHandler, permissionSvc *services.PermissionService) *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)

			protected := auth.Group("")
			protected.Use(middleware.AuthMiddleware(cfg))
			{
				protected.POST("/change-password", authHandler.ChangePassword)
			}
		}

		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(cfg))
		{
			users.GET("", userHandler.SearchUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PATCH("/:id", userHandler.UpdateUser)
			users.PUT("/:id/avatar", userHandler.UpdateAvatar)
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
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg), middleware.RequireSystemPermission(services.SystemPermissionManageRoles, permissionSvc))
		{
			roles := admin.Group("/roles")
			{
				roles.GET("", roleHandler.ListSystemRoles)
				roles.POST("", roleHandler.CreateSystemRole)
				roles.GET("/:roleId", roleHandler.GetRole)
				roles.PUT("/:roleId", roleHandler.UpdateSystemRole)
				roles.DELETE("/:roleId", roleHandler.DeleteRole)
			}

			adminUsers := admin.Group("/users")
			{
				adminUsers.GET("/:userId/roles", roleHandler.GetUserRoles)
				adminUsers.POST("/:userId/roles", roleHandler.AssignUserRoles)
			}

			templates := admin.Group("/project-templates")
			{
				templates.GET("", templateHandler.ListTemplates)
				templates.POST("", templateHandler.CreateTemplate)
			}
		}
	}

	return r
}

