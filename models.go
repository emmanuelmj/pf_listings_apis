// models.go
package main

import "time"

// Project represents a record in the 'approved_projects' table.
type Project struct {
	PID         int       `json:"p_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatorID   int       `json:"creator_id"`
	CreatorName string    `json:"creator_name"`
	StartDate   time.Time `json:"start_date"`
	Status      string    `json:"status"`
}

// BufferProject represents a record in the 'buffer_projects' table.
type BufferProject struct {
	RID         int       `json:"r_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatorID   int       `json:"creator_id"`
	CreatorName string    `json:"creator_name"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// createProjectReq is the JSON body for submitting or creating a project.
type createProjectReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	// CreatorID is now read from the auth context, not the body.
}

// roleChangeReq is the JSON body for assigning/revoking roles.
type roleChangeReq struct {
	UserID   int    `json:"user_id" binding:"required"`
	UserName string `json:"user_name"` // Required for assignment
	Space    string `json:"space" binding:"required"`
}
