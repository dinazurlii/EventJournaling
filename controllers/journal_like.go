package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

func ToggleJournalLike(c *gin.Context) {
	journalID := c.Param("id")
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// cek sudah like atau belum
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM journal_likes
			WHERE user_id = $1 AND journal_id = $2
		)
	`
	_ = config.DB.QueryRow(ctx, checkQuery, userID, journalID).Scan(&exists)

	if exists {
		// UNLIKE
		deleteQuery := `
			DELETE FROM journal_likes
			WHERE user_id = $1 AND journal_id = $2
		`
		_, _ = config.DB.Exec(ctx, deleteQuery, userID, journalID)

		c.JSON(http.StatusOK, gin.H{
			"liked": false,
		})
		return
	}

	// LIKE
	insertQuery := `
		INSERT INTO journal_likes (user_id, journal_id)
		VALUES ($1, $2)
	`
	_, err := config.DB.Exec(ctx, insertQuery, userID, journalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like journal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked": true,
	})
}

func GetJournalLikes(c *gin.Context) {
	journalID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var total int
	query := `
		SELECT COUNT(*)
		FROM journal_likes
		WHERE journal_id = $1
	`
	_ = config.DB.QueryRow(ctx, query, journalID).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"journal_id":  journalID,
		"total_likes": total,
	})
}
