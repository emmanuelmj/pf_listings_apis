// handler_maintainer_delete.go
package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// DELETE /projects/:id/maintainers/:user_id
// deleteMaintainer removes a maintainer from a project.
// Access: SuperAdmin, Admin, or Project Creator
func deleteMaintainer(c *gin.Context) {
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

	// 3. Get User ID (to remove) from URL
	user_id_to_remove, err := getIntParam(c, "user_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id to remove"})
		return
	}

	// 4. --- Permission Check ---
	// Check if the authenticated user has permission (SuperAdmin, Admin, or Creator)
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
		respondErr(c, http.StatusForbidden, "user is not authorized to remove maintainers", nil)
		return
	}

	// 6. --- Permission Granted: Perform Database Action ---
	// Delete the maintainer from the 'maintainers' table
	cmdTag, err := conn.Exec(context.Background(),
		`DELETE FROM maintainers WHERE p_id = $1 AND user_id = $2`,
		p_id, user_id_to_remove)

	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to remove maintainer", err)
		return
	}

	// Check if a row was actually deleted
	if cmdTag.RowsAffected() == 0 {
		respondErr(c, http.StatusNotFound, "maintainer not found for this project", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "maintainer removed",
		"p_id":    p_id,
		"user_id": user_id_to_remove,
	})
}
