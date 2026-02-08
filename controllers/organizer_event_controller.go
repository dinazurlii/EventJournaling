package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

// ===============================
// CREATE ORGANIZER EVENT
// ===============================
func CreateOrganizerEvent(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req struct {
		Title        string    `json:"title" binding:"required"`
		Description  string    `json:"description"`
		StartDate    time.Time `json:"start_date" binding:"required"`
		EndDate      time.Time `json:"end_date" binding:"required"`
		Latitude     float64   `json:"latitude" binding:"required"`
		Longitude    float64   `json:"longitude" binding:"required"`
		LocationName string    `json:"location_name" binding:"required"`
		IsPaid       bool      `json:"is_paid"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
    INSERT INTO events (
        title, description, start_date, end_date,
        latitude, longitude, location_name,
        is_paid, event_type,
        status, created_by
    )
    VALUES (
        $1,$2,$3,$4,
        $5,$6,$7,
        $8,'organizer',
        'pending',
        $9)
    RETURNING id
`
	var eventID int

	err := config.DB.QueryRow(
		context.Background(),
		query,
		req.Title,
		req.Description,
		req.StartDate,
		req.EndDate,
		req.Latitude,
		req.Longitude,
		req.LocationName,
		req.IsPaid,
		userID,
	).Scan(&eventID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         eventID,
		"event_type": "organizer",
		"is_public":  true,
	})
}

// ===============================
// ORGANIZER EVENT DETAIL
// ===============================
func GetOrganizerEventDetail(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT
			id,
			title,
			description,
			event_date,
			latitude,
			longitude,
			location_name,
			created_at
		FROM events
		WHERE id = $1
	`

	var (
		eventID      int
		title        string
		description  string
		eventDate    time.Time
		lat          float64
		lng          float64
		locationName string
		createdAt    time.Time
	)

	err := config.DB.QueryRow(
		context.Background(),
		query,
		id,
	).Scan(
		&eventID,
		&title,
		&description,
		&eventDate,
		&lat,
		&lng,
		&locationName,
		&createdAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            eventID,
		"title":         title,
		"description":   description,
		"start_date":    eventDate,
		"end_date":      nil, // belum disimpan di DB
		"latitude":      lat,
		"longitude":     lng,
		"location_name": locationName,
		"is_paid":       false, // rule di code
		"event_type":    "organizer",
		"is_public":     true,
		"created_at":    createdAt,
	})
}

// ===============================
// SEARCH ORGANIZER EVENTS
// ===============================
func SearchOrganizerEvents(c *gin.Context) {
	start := c.Query("start_date")
	end := c.Query("end_date")

	query := `
        SELECT
            id,
            title,
            start_date,
            end_date,
            latitude,
            longitude,
            location_name
        FROM events
        WHERE event_type = 'organizer'
        AND status = 'published'
        AND start_date >= $1
        AND end_date <= $2
        ORDER BY start_date ASC
    `

	rows, err := config.DB.Query(
		context.Background(),
		query,
		start,
		end,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []gin.H

	for rows.Next() {
		var (
			id           int
			title        string
			startDate    time.Time
			endDate      time.Time
			latitude     float64
			longitude    float64
			locationName string
		)

		err := rows.Scan(
			&id,
			&title,
			&startDate,
			&endDate,
			&latitude,
			&longitude,
			&locationName,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		results = append(results, gin.H{
			"id":            id,
			"title":         title,
			"start_date":    startDate,
			"end_date":      endDate,
			"latitude":      latitude,
			"longitude":     longitude,
			"location_name": locationName,
			"is_paid":       false,
			"event_type":    "organizer",
			"is_public":     true,
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
	})
}
