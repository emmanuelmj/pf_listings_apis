// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/GCET-Open-Source-Foundation/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool" // Recommended for concurrent use
)

// ---------------------- Configuration / Constants ----------------------

const (
	SpaceSuperadmins = "superadmins"
	SpaceAdmins      = "admins"
	SpaceUsers       = "users"
	MemberRole       = "member"
)

// ---------------------- Globals ----------------------

var (
	// Use pgxpool for connection pooling in a real app
	conn *pgxpool.Pool 
)

// ---------------------- Router registration ----------------------

func registerRoutes(r *gin.Engine) {
	// Public routes (no auth required)
	r.GET("/all", getAllProjects)

	// All other routes require at least an authenticated user
	protected := r.Group("/", DummyAuthMiddleware())

	// --- User routes (RequireRole("user")) ---
	// "user" is the base role (creator)
	userRoutes := protected.Group("/", RequireRole("user"))
	{
		// Submit a project to the buffer
		userRoutes.POST("/projects/submit", submitProject)
	}

	// --- Admin routes (RequireRole("admin")) ---
	adminRoutes := protected.Group("/admin", RequireRole("admin"))
	{
		adminRoutes.GET("/pending", getPendingProjects)
		adminRoutes.POST("/approve/:id", approveProject)
		adminRoutes.POST("/reject/:id", rejectProject)
	}

	// --- SuperAdmin routes (RequireRole("superadmin")) ---
	superadminRoutes := protected.Group("/superadmin", RequireRole("superadmin"))
	{
		// Create a project directly, bypassing approval
		superadminRoutes.POST("/create", createProjectAsSuperadmin)
		// Delete an approved project
		superadminRoutes.DELETE("/delete/:id", deleteProjectAsSuperadmin)
	}
}

// ---------------------- Main ----------------------

func main() {
	// Connect to project_forum DB (main app DB)
	connString := os.Getenv("PF_DB_CONN")
	if connString == "" {
		connString = "postgres://postgres:postgres@localhost:5432/project_forum"
	}
	
	var err error
	// Use pgxpool.New for concurrency-safe connection pooling
	conn, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to project DB: %v\n", err)
	}
	defer conn.Close()
	fmt.Println("Connected to project_forum DB")

	// Initialize auth (separate auth DB)
	if err := auth.Init(5432, "postgres", "postgres", "authdb"); err != nil {
		log.Fatalf("auth.Init failed: %v\n", err)
	}
	fmt.Println("auth initialized")

	// Router
	r := gin.Default()
	registerRoutes(r)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
