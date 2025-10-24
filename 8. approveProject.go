// handlers_project_approval.go
package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// GET /admin/pending - list all projects awaiting approval
func getPendingProjects(c *gin.Context) {
	rows, err := conn.Query(context.Background(),
		"SELECT r_id, name, description, creator_id, creator_name, status, submitted_at FROM buffer_projects WHERE status='pending'")
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to fetch pending projects", err)
		return
	}
	defer rows.Close()

	// This struct is in models.go
	var out []BufferProject
	// pgx.AssignRows is a helper to scan all rows into a slice
	if err = pgx.AssignRows(rows, &out); err != nil {
		respondErr(c, http.StatusInternalServerError, "scan failed", err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// POST /admin/approve/:id - approve a pending project
func approveProject(c *gin.Context) {
	rid, err := getIntParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid buffer project id"})
		return
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "tx begin failed", err)
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Move from buffer_projects to approved_projects
	// UPDATED: Status is now set to 'upcoming' instead of 'in_progress'
	if _, err := tx.Exec(context.Background(),
		`INSERT INTO approved_projects (name, description, creator_id, creator_name, start_date, status)
         SELECT name, description, creator_id, creator_name, CURRENT_DATE, 'upcoming' 
         FROM buffer_projects WHERE r_id=$1 AND status='pending'`, rid); err != nil {
		respondErr(c, http.StatusInternalServerError, "approval insert failed", err)
		return
	}

	// 2. Delete from buffer_projects
	cmdTag, err := tx.Exec(context.Background(), `DELETE FROM buffer_projects WHERE r_id=$1`, rid)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "approval delete failed", err)
		return
	}
	if cmdTag.RowsAffected() == 0 {
		respondErr(c, http.StatusNotFound, "project not found or not pending", nil)
		return
	}

	// 3. Commit
	if err := tx.Commit(context.Background()); err != nil {
		respondErr(c, http.StatusInternalServerError, "commit failed", err)
		return
	}

	// Respond with the new status
	c.JSON(http.StatusOK, gin.H{"status": "approved", "new_project_status": "upcoming"})
}

// POST /admin/reject/:id - reject a pending project
func rejectProject(c *gin.Context) {
	rid, err := getIntParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid buffer project id"})
		return
	}

	// This just updates the status in the buffer table, as requested.
	cmdTag, err := conn.Exec(context.Background(),
		`UPDATE buffer_projects SET status='rejected' WHERE r_id=$1 AND status='pending'`, rid)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "reject failed", err)
		return
	}
	if cmdTag.RowsAffected() == 0 {
		respondErr(c, http.StatusNotFound, "project not found or not pending", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}
