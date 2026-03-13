package api

import (
	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/api/middleware"
	"projektus-backend/internal/services"
)

func SetupRouter(cfg *config.Config, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, meetingHandler *handlers.MeetingHandler, roleHandler *handlers.RoleHandler, projectHandler *handlers.ProjectHandler, projectMemberHandler *handlers.ProjectMemberHandler, templateHandler *handlers.TemplateHandler, permissionSvc *services.PermissionService) *gin.Engine {
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

