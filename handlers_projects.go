// handlers_projects.go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// --- Public Handlers ---

// GET /all - list all *approved* projects
func getAllProjects(c *gin.Context) {
	rows, err := conn.Query(context.Background(), "SELECT p_id, name, description, creator_id, creator_name, start_date, status FROM approved_projects")
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to fetch projects", err)
		return
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var sd time.Time
		if err := rows.Scan(&p.PID, &p.Name, &p.Description, &p.CreatorID, &p.CreatorName, &sd, &p.Status); err != nil {
			respondErr(c, http.StatusInternalServerError, "scan failed", err)
			return
		}
		p.StartDate = sd
		out = append(out, p)
	}
	c.JSON(http.StatusOK, out)
}

// --- User Handlers (Creators) ---

// POST /projects/submit - submit a project for approval
func submitProject(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Get creator_id from the authenticated user context
	creatorID, ok := getUserID(c)
	if !ok {
		respondErr(c, http.StatusUnauthorized, "invalid user ID in context", nil)
		return
	}

	// Insert project into the buffer_projects table
	// It defaults to 'pending' status
	row := conn.QueryRow(context.Background(),
		`INSERT INTO buffer_projects (name, description, creator_id, creator_name) 
         VALUES ($1, $2, $3, (SELECT name FROM names WHERE id=$3)) RETURNING r_id`,
		req.Name, req.Description, creatorID)

	var rid int
	if err := row.Scan(&rid); err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to submit project", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"r_id": rid, "status": "pending"})
}

// --- Admin Handlers ---

// GET /admin/pending - list all projects awaiting approval
func getPendingProjects(c *gin.Context) {
	rows, err := conn.Query(context.Background(),
		"SELECT r_id, name, description, creator_id, creator_name, status, submitted_at FROM buffer_projects WHERE status='pending'")
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to fetch pending projects", err)
		return
	}
	defer rows.Close()

	var out []BufferProject
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
	if _, err := tx.Exec(context.Background(),
		`INSERT INTO approved_projects (name, description, creator_id, creator_name, start_date, status)
         SELECT name, description, creator_id, creator_name, CURRENT_DATE, 'in_progress' 
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

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// POST /admin/reject/:id - reject a pending project
func rejectProject(c *gin.Context) {
	rid, err := getIntParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid buffer project id"})
		return
	}

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

// --- SuperAdmin Handlers ---

// POST /superadmin/create - create project directly (bypasses buffer)
func createProjectAsSuperadmin(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Superadmin must provide creator_id in the body for this direct-insert.
	// We will read it from the body, not context.
	var reqWithCreator struct {
		createProjectReq
		CreatorID int `json:"creator_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqWithCreator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creator_id is required for superadmin creation"})
		return
	}

	// Insert project directly into approved_projects
	row := conn.QueryRow(context.Background(),
		`INSERT INTO approved_projects (name, description, creator_id, creator_name, start_date, status) 
         VALUES ($1,$2,$3,(SELECT name FROM names WHERE id=$3), CURRENT_DATE, 'in_progress') RETURNING p_id`,
		reqWithCreator.Name, reqWithCreator.Description, reqWithCreator.CreatorID)

	var pid int
	if err := row.Scan(&pid); err != nil {
		respondErr(c, http.StatusInternalServerError, "insert failed", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"p_id": pid})
}

// DELETE /superadmin/delete/:id - delete an *approved* project
func deleteProjectAsSuperadmin(c *gin.Context) {
	pid, err := getIntParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	tx, err := conn.Begin(context.Background())
	if err !=. nil {
		respondErr(c, http.StatusInternalServerError, "tx begin failed", err)
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Archive to deleted_projects
	if _, err := tx.Exec(context.Background(),
		`INSERT INTO deleted_projects (p_id, name, description, creator_id, creator_name) 
         SELECT p_id, name, description, creator_id, creator_name 
         FROM approved_projects WHERE p_id=$1`, pid); err != nil {
		respondErr(c, http.StatusInternalServerError, "archive failed", err)
		return
	}

	// 2. Delete from approved_projects (cascades to maintainers/contributors)
	cmdTag, err := tx.Exec(context.Background(), `DELETE FROM approved_projects WHERE p_id=$1`, pid)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "delete failed", err)
		return
	}
	if cmdTag.RowsAffected() == 0 {
		respondErr(c, http.StatusNotFound, "project not found", nil)
		return
	}

	// 3. Commit
	if err := tx.Commit(context.Background()); err != nil {
		respondErr(c, http.StatusInternalServerError, "commit failed", err)
		return
	}

	c.Status(http.StatusNoContent)
}
