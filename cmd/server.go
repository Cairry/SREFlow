package main

import (
	"SREFlow/config"
	"SREFlow/internal/ctx"
	"SREFlow/internal/middleware"
	"SREFlow/internal/repos"
	"SREFlow/internal/routes"
	"SREFlow/internal/services"
	"SREFlow/pkg/client"
	"SREFlow/pkg/license"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.Init()

	// Validate license
	validateLicense()

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

	// Start license expiry monitor
	go monitorLicenseExpiry(config.Application.License.Path)

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

func validateLicense() {
	licCfg := config.Application.License
	if licCfg.Path == "" {
		log.Fatal("[License] License path is not configured")
	}

	lic, err := license.VerifyFromFile(licCfg.Path)
	if err != nil {
		log.Fatalf("[License] Validation failed: %v", err)
	}

	if lic.MachineID != "" {
		uuid, err := os.ReadFile("/sys/class/dmi/id/product_uuid")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if string(uuid) != lic.MachineID {
			fmt.Fprintf(os.Stderr, "Error: Machine ID does not match\n")
			os.Exit(1)
		}
	}

	fmt.Printf("[License] Valid | Holder: %s\n", lic.Holder)
	if lic.ExpiresAt > 0 {
		fmt.Printf("[License] Expires at: %s\n", time.Unix(lic.ExpiresAt, 0).Format("2006-01-02 15:04:05"))
	}
}

func monitorLicenseExpiry(licensePath string) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		lic, err := license.VerifyFromFile(licensePath)
		if err != nil {
			log.Fatalf("[License] Check failed: %v", err)
		}
		if lic.IsExpired() {
			panic(fmt.Sprintf("[License] Expired! Holder: %s, Expired at: %s", lic.Holder, time.Unix(lic.ExpiresAt, 0).Format("2006-01-02 15:04:05")))
		}
	}
}
