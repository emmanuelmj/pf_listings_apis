package main

// StatusUpdateRequest is the expected JSON payload for updating a project's status
type StatusUpdateRequest struct {
	ProjectID int    `json:"p_id"` // The approved_projects.p_id
	NewStatus string `json:"new_status"` // Must be 'in_progress', 'completed', or 'upcoming'
}

// ... existing models
// UpdateProjectStatusHandler handles the request to change the status of an approved project.
func UpdateProjectStatusHandler(c *gin.Context) {
    // 1. Get the current user's ID and Role from the Gin Context (provided by AuthRequired middleware)
    currentUserID, exists := c.Get("userID") // Assuming userID is stored as an interface{}
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token context"})
        return
    }
    
    // Assuming the role is also stored in the context by the middleware
    currentUserRole, exists := c.Get("userRole") 
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in token context"})
        return
    }
    role := currentUserRole.(string) // Cast the role to string
    userID := currentUserID.(int) // Cast the ID to int

    // 2. Decode the request body
    var req StatusUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
        return
    }
    
    // 3. Status Validation
    validStatuses := map[string]bool{"in_progress": true, "completed": true, "upcoming": true}
    if _, ok := validStatuses[req.NewStatus]; !ok {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value. Must be 'in_progress', 'completed', or 'upcoming'."})
        return
    }

    // 4. Role & Permission Check
    
    // Admins/Super Admins have universal rights.
    if role == "superadmin" || role == "admin" {
        // Admins and Superadmins are allowed to update any project's status.
    } else if role == "creator" || role == "maintainer" {
        // Creators and Maintainers can only update *their own* project.
        var projectCreatorID int
        err := db.QueryRow("SELECT creator_id FROM approved_projects WHERE p_id = $1", req.ProjectID).Scan(&projectCreatorID)
        
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "Project not found."})
            return
        }
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking project creator"})
            return
        }
        
        // A Creator can delete their project[cite: 26], and Maintainers have the same allowances 
        // as the Creator except deletion[cite: 27], so both can update status.
        if projectCreatorID != userID {
            c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. Only the Creator/Maintainer or a higher role can update this project's status."})
            return
        }
    } else {
        // All other roles (Contributor, Viewer) are forbidden.
        c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied. You must be a Creator, Maintainer, Admin, or Superadmin to update project status."})
        return
    }

    // 5. Execute the Update
    result, err := db.Exec(`
        UPDATE approved_projects 
        SET status = $1
        WHERE p_id = $2
    `, req.NewStatus, req.ProjectID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project status: " + err.Error()})
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or status is already the same."})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Project status updated successfully to " + req.NewStatus})
}