// handlers_roles.go

// ... (existing assignSuperAdmin function) ...

// POST /superadmin/roles/admin - Assign admin role
// Only accessible by existing Superadmins.
func assignAdmin(c *gin.Context) {
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
	// and call auth.Create_permissions for the 'admins' space.
	err := AssignMemberToSpace(req.UserID, req.UserName, SpaceAdmins)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, "failed to assign admin role", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin role assigned successfully",
		"user_id": req.UserID,
		"space":   SpaceAdmins,
	})
}