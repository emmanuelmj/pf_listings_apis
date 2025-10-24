// auth.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/GCET-Open-Source-Foundation/auth"
	"github.com/gin-gonic/gin"
)

// ---------------------- Auth helpers (use ONLY real auth functions) ----------------------

// HasRole implements the hierarchy based on your new rules.
// - "superadmin": checks SpaceSuperadmins
// - "admin": checks ONLY SpaceAdmins
// - "user": allows any authenticated user
func HasRole(userIDStr string, require string) bool {
	if require == "superadmin" {
		return auth.Check_permissions(userIDStr, SpaceSuperadmins, MemberRole)
	}
	if require == "admin" {
		// UPDATED: Only checks SpaceAdmins, does not include superadmin.
		return auth.Check_permissions(userIDStr, SpaceAdmins, MemberRole)
	}
	if require == "user" {
		// "user" or any other - treat as allowed for any authenticated user
		return true
	}
	// Default to false if role not recognized
	return false
}

// AssignMemberToSpace assigns (creates) a membership using auth.Create_permissions.
func AssignMemberToSpace(userID int, userName, space string) error {
	// Upsert into local names table (domain user store)
	if _, err := conn.Exec(context.Background(),
		`INSERT INTO names (id, name) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name`,
		userID, userName); err != nil {
		return fmt.Errorf("upsert names failed: %w", err)
	}

	// Create permission via auth library.
	if err := auth.Create_permissions(uidToStr(userID), space, MemberRole); err != nil {
		return fmt.Errorf("auth.Create_permissions failed: %w", err)
	}
	return nil
}

// RemoveMemberFromSpace revokes membership using auth.Delete_permission.
func RemoveMemberFromSpace(userID int, space string) error {
	if err := auth.Delete_permission(uidToStr(userID), space, MemberRole); err != nil {
		return fmt.Errorf("auth.Delete_permission failed: %w", err)
	}
	return nil
}

// ---------------------- Middleware ----------------------

// DummyAuthMiddleware placeholder
func DummyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetHeader("X-Dummy-User")
		c.Set("user_id", uid)
		
		// Also fetch and set the user's ID as an integer for handlers
		uidInt, _ := strconv.Atoi(uid)
		c.Set("user_id_int", uidInt)

		c.Next()
	}
}

// RequireRole middleware uses the updated HasRole()
func RequireRole(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
			c.Abort()
			return
		}
		userIDStr := fmt.Sprintf("%v", val)
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			c.Abort()
			return
		}
		if !HasRole(userIDStr, required) {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied for this role"})
			c.Abort()
			return
		}
		c.Next()
	}
}
