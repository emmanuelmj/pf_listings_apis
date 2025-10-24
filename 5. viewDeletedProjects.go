// handlers_deleted.go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// --- Models ---

// DeletedProject represents a record in the 'deleted_projects' table.
type DeletedProject struct {
	PID          int       `json:"p_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CreatorID    int       `json:"creator_id"`
	CreatorName  string    `json:"creator_name"`
	DeletedDate  time.Time `json:"deleted_date"`
}

// --- Handlers ---

// getAllDeletedProjects handles viewing all deleted projects.
// Access: Admin, SuperAdmin
func getAllDeletedProjects(c *gin.Context) {
	rows, err := conn.Query(context.Background(),
		"SELECT p_id, name, description, creator_id, creator_name, deleted_date FROM deleted_projects")
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to fetch deleted projects", err)
		return
	}
	defer rows.Close()

	var out []DeletedProject
	if err = pgx.AssignRows(rows, &out); err != nil {
		respondErr(c, http.StatusInternalServerError, "scan failed", err)
		return
	}

	c.JSON(http.StatusOK, out)
}

// getMyDeletedProjects handles viewing the user's own deleted projects.
// Access: User (Creator)
func getMyDeletedProjects(c *gin.Context) {
	// Get creator_id from the authenticated user context
	creatorID, ok := getUserID(c)
	if !ok {
		respondErr(c, http.StatusUnauthorized, "invalid user ID in context", nil)
		return
	}

	rows, err := conn.Query(context.Background(),
		"SELECT p_id, name, description, creator_id, creator_name, deleted_date FROM deleted_projects WHERE creator_id=$1", creatorID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to fetch your deleted projects", err)
		return
	}
	defer rows.Close()

	var out []DeletedProject
	if err = pgx.AssignRows(rows, &out); err != nil {
		respondErr(c, http.StatusInternalServerError, "scan failed", err)
		return
	}

	c.JSON(http.StatusOK, out)
}
