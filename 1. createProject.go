package pfapi

// ... (other imports) ...
import {
"fmt" // <-- Make sure this import is present
"github.com/GCET-Open-Source-Foundation/auth" 
}// <-- Make sure this import is present

// ... (getAllProjects, getPendingProjects, etc.) ...

// POST /projects - A unified endpoint to create or submit a project.
// - Admins/Superadmins: Auto-approved, inserts into 'approved_projects'.
// - Users/Creators: Submitted for review, inserts into 'buffer_projects'.
func createProject(c *gin.Context) {
	// 1. Get user ID (int) from context
	creatorID, ok := getUserID(c)
	if !ok {
		respondErr(c, http.StatusUnauthorized, "invalid user ID in context", nil)
		return
	}

	// 2. Get user ID (string) for auth checks
	// We know "user_id" string exists from the middleware
	val, _ := c.Get("user_id")
	creatorIDStr := fmt.Sprintf("%v", val)

	// 3. Bind the request body (uses 'createProjectReq' from models.go)
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// 4. Check roles using the auth library
	isSuperAdmin := auth.Check_permissions(creatorIDStr, SpaceSuperadmins, MemberRole)
	isAdmin := auth.Check_permissions(creatorIDStr, SpaceAdmins, MemberRole)

	// 5. Execute logic based on role
	if isSuperAdmin || isAdmin {
		// --- AUTO-APPROVE Logic (for Superadmin/Admin) ---
		// Insert directly into approved_projects
		row := conn.QueryRow(context.Background(),
			`INSERT INTO approved_projects (name, description, creator_id, creator_name, start_date, status) 
			 VALUES ($1,$2,$3,(SELECT name FROM names WHERE id=$3), CURRENT_DATE, 'in_progress') RETURNING p_id`,
			req.Name, req.Description, creatorID)

		var pid int
		if err := row.Scan(&pid); err != nil {
			respondErr(c, http.StatusInternalServerError, "direct insert failed", err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"p_id": pid, "status": "approved"})

	} else {
		// --- SUBMIT-TO-BUFFER Logic (for Creator/User) ---
		// Insert into buffer_projects
		row := conn.QueryRow(context.Background(),
			`INSERT INTO buffer_projects (name, description, creator_id, creator_name) 
			 VALUES ($1, $2, $3, (SELECT name FROM names WHERE id=$3)) RETURNING r_id`,
			req.Name, req.Description, creatorID)

		var rid int
		if err := row.Scan(&rid); err != nil {
			respondErr(c, http.StatusInternalServerError, "failed to submit project", err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"r_id": rid, "status": "pending"})
	}
}

// ... (other handlers) ...

/*
NOTE:
You can now REMOVE the old handlers `submitProject` and `createProjectAsSuperadmin`
from this file, as this new `createProject` function replaces them both.
*/