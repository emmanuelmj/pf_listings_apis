// handlers_project_deletion.go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/GCET-Open-Source-Foundation/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// DELETE /projects/:id - A unified endpoint to delete a project.
// - Admins/Superadmins: Can delete any project.
// - Users/Creators: Can only delete their own projects.
func deleteProject(c *gin.Context) {
	// 1. Get project ID from path
	pid, err := getIntParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	// 2. Get user's ID (int and string)
	userID, ok := getUserID(c)
	if !ok {
		respondErr(c, http.StatusUnauthorized, "invalid user ID in context", nil)
		return
	}
	val, _ := c.Get("user_id")
	userIDStr := fmt.Sprintf("%v", val)

	// 3. Get user's roles
	isSuperAdmin := auth.Check_permissions(userIDStr, SpaceSuperadmins, MemberRole)
	isAdmin := auth.Check_permissions(userIDStr, SpaceAdmins, MemberRole)

	// 4. Find the project's creator to check ownership
	var projectCreatorID int
	err = conn.QueryRow(context.Background(),
		`SELECT creator_id FROM approved_projects WHERE p_id=$1`, pid).Scan(&projectCreatorID)

	if err == pgx.ErrNoRows {
		respondErr(c, http.StatusNotFound, "project not found", nil)
		return
	}
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to check project ownership", err)
		return
	}

	// 5. Check permissions
	// User can delete if they are a Superadmin, an Admin, OR if they are the creator.
	canDelete := isSuperAdmin || isAdmin || (userID == projectCreatorID)

	if !canDelete {
		respondErr(c, http.StatusForbidden, "you do not have permission to delete this project", nil)
		return
	}

	// 6. Proceed with deletion (Archive and Delete)
	tx, err := conn.Begin(context.Background())
	if err != nil {
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
		// This should not happen if we found it earlier, but it's a good safeguard.
		respondErr(c, http.StatusNotFound, "project not found during delete", nil)
		return
	}

	// 3. Commit
	if err := tx.Commit(context.Background()); err != nil {
		respondErr(c, http.StatusInternalServerError, "commit failed", err)
		return
	}

	c.Status(http.StatusNoContent)
}