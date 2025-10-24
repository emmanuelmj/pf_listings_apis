// handlers_roles.go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST /superadmin/roles/superadmin - Assign superadmin role
// Only accessible by existing Superadmins.
func assignSuperAdmin(c *gin.Context) {
	var req assignUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// UserName is required to ensure the 'names' table is populated
	if req.UserName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_name is required"})
		return
	}

	// Use the helper from auth.go to update the 'names' table
	// and call auth.Create_permissions.
	err := AssignMemberToSpace(req.UserID, req.UserName, SpaceSuperadmins)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to assign superadmin role", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Superadmin role assigned successfully",
		"user_id": req.UserID,
		"space":   SpaceSuperadmins,
	})
}
