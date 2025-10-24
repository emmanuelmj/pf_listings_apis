// handlers_roles.go

// ... (existing assignSuperAdmin and assignAdmin functions) ...

// DELETE /superadmin/roles/admin - Revoke admin role
// Only accessible by existing Superadmins.
func revokeAdmin(c *gin.Context) {
	var req revokeUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body, user_id is required"})
		return
	}

	// Use the helper from auth.go to call auth.Delete_permission
	// for the 'admins' space.
	err := RemoveMemberFromSpace(req.UserID, SpaceAdmins)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to revoke admin role", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin role revoked successfully",
		"user_id": req.UserID,
		"space":   SpaceAdmins,
	})
}