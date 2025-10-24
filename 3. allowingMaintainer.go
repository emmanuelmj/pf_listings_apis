// handlers_roles.go
package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- Models ---

// addMaintainerReq is the JSON body for adding a maintainer.
type addMaintainerReq struct {
	UserID   int    `json:"user_id" binding:"required"`
	UserName string `json:"user_name" binding:"required"` // Name for the 'm_name' field
}

// --- Handlers ---

// POST /projects/:id/maintainers
// addMaintainer adds a user as a maintainer for a specific project.
// Access: SuperAdmin, Admin, or Project Creator
func addMaintainer(c *gin.Context) {
	// 1. Get Authenticated User
	authedUserID, ok := getUserID(c)
	if !ok {
		respondErr(c, http.StatusUnauthorized, "invalid user ID in context", nil)
		return
	}

	// 2. Get Project ID from URL
	p_id, err := getIntParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	// 3. Bind Request Body
	var req addMaintainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body, user_id and user_name are required"})
		return
	}

	// 4. --- Permission Check ---
	authedUserIDStr := uidToStr(authedUserID)
	isSuperAdmin := HasRole(authedUserIDStr, "superadmin")
	isAdmin := HasRole(authedUserIDStr, "admin")
	isCreator := false

	// If not a global admin, check if they are the project creator
	if !isSuperAdmin && !isAdmin {
		var creatorID int
		err := conn.QueryRow(context.Background(),
			"SELECT creator_id FROM approved_projects WHERE p_id = $1", p_id).Scan(&creatorID)

		if err == pgx.ErrNoRows {
			respondErr(c, http.StatusNotFound, "project not found", nil)
			return
		}
		if err != nil {
			respondErr(c, http.StatusInternalServerError, "failed to check project ownership", err)
			return
		}
		if creatorID == authedUserID {
			isCreator = true
		}
	}

	// 5. Enforce Permission
	if !isSuperAdmin && !isAdmin && !isCreator {
		respondErr(c, http.StatusForbidden, "user is not authorized to add maintainers", nil)
		return
	}

	// 6. --- Permission Granted: Perform Database Actions ---
	
	tx, err := conn.Begin(context.Background())
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "tx begin failed", err)
		return
	}
	defer tx.Rollback(context.Background())

	// a) Upsert the user into the 'names' table to ensure they exist
	if _, err := tx.Exec(context.Background(),
		`INSERT INTO names (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name`,
		req.UserID, req.UserName); err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to upsert user in names table", err)
		return
	}

	// b) Insert into 'maintainers' table
	row := tx.QueryRow(context.Background(),
		`INSERT INTO maintainers (p_id, user_id, m_name) 
         VALUES ($1, $2, $3) 
         RETURNING m_id`,
		p_id, req.UserID, req.UserName)

	var m_id int
	if err := row.Scan(&m_id); err != nil {
		// Check for unique constraint violation (user already a maintainer)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			respondErr(c, http.StatusConflict, "user is already a maintainer for this project", err)
			return
		}
		respondErr(c, http.StatusInternalServerError, "failed to add maintainer", err)
		return
	}
	
	// c) Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		respondErr(c, http.StatusInternalServerError, "tx commit failed", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"m_id":    m_id,
		"p_id":    p_id,
		"user_id": req.UserID,
		"status":  "maintainer added",
	})
}
