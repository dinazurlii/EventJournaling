package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

type BookmarkInput struct {
	JournalID int `json:"journal_id"`
}

func BookmarkJournal(c *gin.Context) {
	var input BookmarkInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO bookmarks (user_id, journal_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	_, err := config.DB.Exec(ctx, query, userID, input.JournalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to bookmark"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "journal bookmarked",
	})
}

func UnbookmarkJournal(c *gin.Context) {
	journalID := c.Param("journal_id")
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		DELETE FROM bookmarks
		WHERE user_id = $1 AND journal_id = $2
	`

	_, err := config.DB.Exec(ctx, query, userID, journalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unbookmark"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "bookmark removed",
	})
}

func GetMyBookmarks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT j.id, j.title, j.content, j.latitude, j.longitude, j.created_at
		FROM bookmarks b
		JOIN journals j ON j.id = b.journal_id
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC
	`

	rows, err := config.DB.Query(ctx, query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookmarks"})
		return
	}
	defer rows.Close()

	var journals []gin.H

	for rows.Next() {
		var id int
		var title, content string
		var lat, lng float64
		var createdAt time.Time

		if err := rows.Scan(&id, &title, &content, &lat, &lng, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read bookmark"})
			return
		}

		journals = append(journals, gin.H{
			"id":         id,
			"title":      title,
			"content":    content,
			"latitude":   lat,
			"longitude":  lng,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": journals,
	})
}
