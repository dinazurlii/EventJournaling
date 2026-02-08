package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

func GetMapJournals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, title, content, latitude, longitude
		FROM journals
		WHERE is_public = true
		  AND latitude IS NOT NULL
		  AND longitude IS NOT NULL
		ORDER BY created_at DESC
	`

	rows, err := config.DB.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch map journals"})
		return
	}
	defer rows.Close()

	var journals []gin.H

	for rows.Next() {
		var (
			id      int
			title   string
			content string
			lat     float64
			lng     float64
		)

		if err := rows.Scan(&id, &title, &content, &lat, &lng); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read journal"})
			return
		}

		preview := content
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}

		journals = append(journals, gin.H{
			"id":        id,
			"title":     title,
			"preview":   preview,
			"latitude":  lat,
			"longitude": lng,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": journals,
	})
}
