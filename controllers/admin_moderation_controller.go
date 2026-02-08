package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

func GetPendingEvents(c *gin.Context) {
	role := c.GetString("role")

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin only",
		})
		return
	}

	query := `
		SELECT
			e.id,
			e.title,
			e.start_date,
			e.end_date,
			e.location_name,
			u.id,
			u.email,
			e.created_at
		FROM events e
		JOIN users u ON u.id = e.created_by
		WHERE e.status = 'pending'
		ORDER BY e.created_at ASC
	`

	rows, err := config.DB.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var events []gin.H

	for rows.Next() {
		var (
			id           int
			title        string
			startDate    time.Time
			endDate      time.Time
			locationName string
			isPaid       bool
			userID       int
			email        string
			createdAt    time.Time
		)

		if err := rows.Scan(
			&id,
			&title,
			&startDate,
			&endDate,
			&locationName,
			&isPaid,
			&userID,
			&email,
			&createdAt,
		); err != nil {
			continue
		}

		events = append(events, gin.H{
			"id":            id,
			"title":         title,
			"start_date":    startDate,
			"end_date":      endDate,
			"location_name": locationName,
			"is_paid":       isPaid,
			"created_at":    createdAt,
			"creator": gin.H{
				"id":    userID,
				"email": email,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": events,
	})
}

func ApproveEvent(c *gin.Context) {
	role := c.GetString("role")
	eventID := c.Param("id")

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin only",
		})
		return
	}

	query := `
		UPDATE events
		SET status = 'published'
		WHERE id = $1
		  AND event_type = 'organizer'
		  AND status = 'pending'
	`

	cmd, err := config.DB.Exec(
		context.Background(),
		query,
		eventID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found or already processed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event approved",
	})
}

func RejectEvent(c *gin.Context) {
	role := c.GetString("role")
	eventID := c.Param("id")

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin only",
		})
		return
	}

	query := `
		UPDATE events
		SET status = 'rejected'
		WHERE id = $1
		  AND event_type = 'organizer'
		  AND status = 'pending'
	`

	cmd, err := config.DB.Exec(
		context.Background(),
		query,
		eventID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found or already processed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event rejected",
	})
}
