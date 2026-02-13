package controllers

import (
	"context"
	"event-journal-backend/config"
	"event-journal-backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SaveFCMToken(c *gin.Context) {
	userID := c.GetInt("user_id")

	var body struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}

	err := services.SaveUserFCMToken(userID, body.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token saved"})
}

func SaveUserFCMToken(userID int, token string) error {
	query := `
	UPDATE users
	SET fcm_token = $1
	WHERE id = $2
	`
	_, err := config.DB.Exec(context.Background(), query, token, userID)
	return err
}
