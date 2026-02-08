package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

type CreateCommentInput struct {
	Content string `json:"content"`
}

func CreateComment(c *gin.Context) {
	journalID := c.Param("id")
	userID, _ := c.Get("user_id")

	var input CreateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO comments (journal_id, user_id, content)
		VALUES ($1, $2, $3)
	`

	_, err := config.DB.Exec(ctx, query, journalID, userID, input.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "comment created",
	})
}

func GetJournalComments(c *gin.Context) {
	journalID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			c.id,
			c.content,
			c.created_at,
			u.id,
			u.email
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.journal_id = $1
		ORDER BY c.created_at ASC
	`

	rows, err := config.DB.Query(ctx, query, journalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}
	defer rows.Close()

	var comments []gin.H

	for rows.Next() {
		var commentID, userID int
		var content, email string
		var createdAt time.Time

		if err := rows.Scan(&commentID, &content, &createdAt, &userID, &email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read comment"})
			return
		}

		comments = append(comments, gin.H{
			"id":         commentID,
			"content":    content,
			"created_at": createdAt,
			"user": gin.H{
				"id":    userID,
				"email": email,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": comments,
	})
}

func DeleteComment(c *gin.Context) {
	commentID := c.Param("id")
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		DELETE FROM comments
		WHERE id = $1 AND user_id = $2
	`

	result, err := config.DB.Exec(ctx, query, commentID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment deleted",
	})
}
