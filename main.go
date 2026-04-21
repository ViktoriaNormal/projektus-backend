package main

import (
	"log"

	_ "github.com/lib/pq"

	"projektus-backend/config"
	"projektus-backend/internal/api"
	"projektus-backend/internal/bootstrap"
)

// main — точка входа. Всё реальное связывание зависимостей живёт в
// internal/bootstrap; здесь только config → App → Router.
func main() {
	cfg := config.Load()

	app, err := bootstrap.NewApp(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer app.Conn.Close()

	h := app.Handlers
	router := api.SetupRouter(
		cfg,
		h.Auth, h.User, h.Notification, h.Meeting, h.Role, h.Project,
		h.ProjectMember, h.Template, h.Board, h.Task, h.Sprint,
		h.ProductBacklog, h.SprintBacklog, h.AdminUser, h.AdminPasswordPolicy,
		h.ProjectRole, h.ProjectParam, h.Tag,
		h.ScrumAnalytics, h.KanbanAnalytics,
		app.Services.Project, app.Services.Permission,
	)

	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
