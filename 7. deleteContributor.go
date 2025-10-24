package main

// ContributorRequest is used for both adding and removing a contributor
type ContributorRequest struct {
	ProjectID int    `json:"p_id"`    // The approved_projects.p_id
	UserID    int    `json:"user_id"` // The names.id of the user to be removed
}

// ... existing models
// RemoveContributorHandler handles the request to remove a user as a contributor from a project.
func RemoveContributorHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Get the current user's ID from the JWT token
    currentUserID, ok := r.Context().Value("userID").(int)
    if !ok {
        respondWithError(w, http.StatusUnauthorized, "User ID not found in token context")
        return
    }

    // 2. Decode the request body (using JSON for DELETE is non-standard but common in REST APIs for complex IDs)
    var req ContributorRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
    
    // 3. Permission Check: Verify the current user is the Creator or a Maintainer of the project
    
    // Get the project's Creator ID from approved_projects table
    var creatorID int
    err := db.QueryRow("SELECT creator_id FROM approved_projects WHERE p_id = $1", req.ProjectID).Scan(&creatorID)
    if err != nil {
        if err == sql.ErrNoRows {
            respondWithError(w, http.StatusNotFound, "Project not found in approved list")
        } else {
            respondWithError(w, http.StatusInternalServerError, "Database error checking project creator")
        }
        return
    }

    // Check if the current user is the Creator
    isCreator := currentUserID == creatorID
    
    // Check if the current user is a Maintainer for this project
    var isMaintainer bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM maintainers WHERE p_id = $1 AND user_id = $2)", 
        req.ProjectID, currentUserID).Scan(&isMaintainer)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Database error checking maintainer status")
        return
    }

    // Enforce Role Rule: Only Creator or Maintainer can remove a contributor
    if !isCreator && !isMaintainer {
        respondWithError(w, http.StatusForbidden, "Only the Project Creator or a Maintainer can remove a contributor.")
        return
    }

    // 4. Delete from the contributors table
    result, err := db.Exec(`
        DELETE FROM contributors 
        WHERE p_id = $1 AND user_id = $2
    `, req.ProjectID, req.UserID)

    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to remove contributor: "+err.Error())
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Database error checking affected rows")
        return
    }

    if rowsAffected == 0 {
        respondWithError(w, http.StatusNotFound, "User is not a contributor for this project.")
        return
    }

    respondWithJSON(w, http.StatusOK, map[string]string{
        "message": "Contributor removed successfully.",
    })
}
