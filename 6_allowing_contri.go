package main

// ContributorRequest is the expected JSON payload for adding a contributor
type ContributorRequest struct {
	ProjectID int    `json:"p_id"`    // The approved_projects.p_id
	UserID    int    `json:"user_id"` // The names.id of the user to be added
}

// ... existing models (ApprovedProject, PastProject, etc.)
// AllowContributorHandler handles the request to add a user as a contributor to a project.
func AllowContributorHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Get the current user's ID from the JWT token (provided by AuthRequired middleware)
    // Assuming AuthRequired extracts and stores the ID in the request context
    currentUserID, ok := r.Context().Value("userID").(int)
    if !ok {
        respondWithError(w, http.StatusUnauthorized, "User ID not found in token context")
        return
    }

    // 2. Decode the request body
    var req ContributorRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
    
    // Check if user to be added exists (Optional but good practice)
    var contributorName string
    err := db.QueryRow("SELECT name FROM names WHERE id = $1", req.UserID).Scan(&contributorName)
    if err != nil {
        if err == sql.ErrNoRows {
            respondWithError(w, http.StatusBadRequest, "User to be added does not exist")
        } else {
            respondWithError(w, http.StatusInternalServerError, "Database error checking contributor")
        }
        return
    }

    // 3. Permission Check: Verify the current user is the Creator or a Maintainer of the project
    
    // Get the project's Creator ID from approved_projects table
    var creatorID int
    err = db.QueryRow("SELECT creator_id FROM approved_projects WHERE p_id = $1", req.ProjectID).Scan(&creatorID)
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

    // Enforce Role Rule: Only Creator or Maintainer can allow a contributor 
    if !isCreator && !isMaintainer {
        respondWithError(w, http.StatusForbidden, "Only the Project Creator or a Maintainer can allow a contributor.")
        return
    }

    // 4. Insert into contributors table
    insertQuery := `
        INSERT INTO contributors (p_id, user_id, c_name)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, p_id) DO NOTHING 
        RETURNING c_id
    `
    var cID int
    err = db.QueryRow(insertQuery, req.ProjectID, req.UserID, contributorName).Scan(&cID)

    if err != nil && err != sql.ErrNoRows {
        // sql.ErrNoRows is returned by QueryRow on ON CONFLICT DO NOTHING when no row is returned (i.e., conflict occurred)
        // We handle the conflict implicitly by checking cID later if we want a more specific response.
        respondWithError(w, http.StatusInternalServerError, "Failed to add contributor: "+err.Error())
        return
    }
    
    if cID == 0 {
        respondWithJSON(w, http.StatusOK, map[string]string{"message": "User is already a contributor on this project."})
        return
    }

    respondWithJSON(w, http.StatusCreated, map[string]interface{}{
        "message": "Contributor added successfully",
        "contributor_id": cID,
    })
}