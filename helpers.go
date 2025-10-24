// helpers.go
package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// respondErr logs the error and sends a JSON error message.
func respondErr(c *gin.Context, code int, msg string, err error) {
	if err != nil {
		log.Printf("Error: %s: %v\n", msg, err)
	} else {
		log.Printf("Error: %s\n", msg)
	}
	c.JSON(code, gin.H{"error": msg})
}

// uidToStr converts an integer user ID to a string.
func uidToStr(id int) string {
	return strconv.Itoa(id)
}

// getIntParam fetches an integer path parameter (e.g., :id).
func getIntParam(c *gin.Context, name string) (int, error) {
	p := c.Param(name)
	return strconv.Atoi(p)
}

// getUserID gets the authenticated user's ID (as an int) from the context.
func getUserID(c *gin.Context) (int, bool) {
	val, exists := c.Get("user_id_int")
	if !exists {
		return 0, false
	}
	uid, ok := val.(int)
	return uid, ok
}
