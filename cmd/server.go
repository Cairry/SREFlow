package main

import (
	"RCAgent/config"
	"RCAgent/internal/ctx"
	"RCAgent/internal/middleware"
	"RCAgent/internal/repos"
	"RCAgent/internal/routes"
	"RCAgent/internal/services"
	"RCAgent/pkg/client"
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.Init()

	// Initialize database
	db := client.NewDBClient(config.Application.Database)
	if db == nil {
		log.Fatal("Failed to initialize database")
	}

	// Initialize repositories
	repos := repos.NewEntryRepo(db)

	ctx := ctx.NewContext(context.Background(), repos, nil)

	// Initialize services
	services.Init(ctx)

	// Initialize Gin router
	if config.Application.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Global CORS middleware — must be before any routes to handle preflight OPTIONS
	r.Use(middleware.Cors())

	// Register routes
	routes.Register(r)

	// Start server
	port := config.Application.Server.Port
	if port == "" {
		port = "8080"
	}

	fmt.Printf("[Server] Starting on port %s\n", port)
	fmt.Printf("[Server] Database: %s\n", config.Application.Database.Type)
	fmt.Printf("[Server] API: http://localhost:%s/api\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
