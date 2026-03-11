package api

import (
	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/api/middleware"
)

func SetupRouter(cfg *config.Config, authHandler *handlers.AuthHandler) *gin.Engine {
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
	}

	return r
}

