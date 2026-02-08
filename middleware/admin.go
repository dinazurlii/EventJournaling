package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		userID, _ := c.Get("user_id")

		println("ADMIN CHECK | user_id:", userID, "role:", role)

		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "admin access only",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
