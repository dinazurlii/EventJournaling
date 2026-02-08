package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

type CreateEventInput struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	EventDate    string  `json:"event_date"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	LocationName string  `json:"location_name"`
}

func CreateEvent(c *gin.Context) {
	var input CreateEventInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO events (
			title, description, event_date,
			latitude, longitude, location_name, created_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id
	`

	var eventID int
	err := config.DB.QueryRow(
		ctx,
		query,
		input.Title,
		input.Description,
		input.EventDate,
		input.Latitude,
		input.Longitude,
		input.LocationName,
		userID,
	).Scan(&eventID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      eventID,
		"message": "event created",
	})
}

func GetMyEvents(c *gin.Context) {
	userID, _ := c.Get("user_id")

	query := `
		SELECT id, title, event_date, latitude, longitude, location_name
		FROM events
		WHERE created_by = $1
		ORDER BY event_date DESC
	`

	rows, err := config.DB.Query(context.Background(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch events"})
		return
	}
	defer rows.Close()

	var events []gin.H

	for rows.Next() {
		var id int
		var title, location string
		var date time.Time
		var lat, lng float64

		rows.Scan(&id, &title, &date, &lat, &lng, &location)

		events = append(events, gin.H{
			"id":        id,
			"title":     title,
			"date":      date,
			"latitude":  lat,
			"longitude": lng,
			"location":  location,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": events,
	})
}

func GetEventDetail(c *gin.Context) {
	eventID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ===== GET EVENT =====
	var event struct {
		ID           int
		LocationName string
		Latitude     float64
		Longitude    float64
	}

	eventQuery := `
		SELECT id, location_name, latitude, longitude
		FROM events
		WHERE id = $1
	`

	err := config.DB.QueryRow(ctx, eventQuery, eventID).
		Scan(&event.ID, &event.LocationName, &event.Latitude, &event.Longitude)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	// ===== GET JOURNALS =====
	journalQuery := `
		SELECT id, title, content, created_at
		FROM journals
		WHERE event_id = $1
		ORDER BY created_at DESC
	`

	rows, err := config.DB.Query(ctx, journalQuery, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch journals"})
		return
	}
	defer rows.Close()

	var journals []gin.H

	for rows.Next() {
		var id int
		var title, content string
		var createdAt time.Time

		if err := rows.Scan(&id, &title, &content, &createdAt); err != nil {
			continue
		}

		journals = append(journals, gin.H{
			"id":         id,
			"title":      title,
			"content":    content,
			"created_at": createdAt,
		})
	}

	// ===== RESPONSE =====
	c.JSON(http.StatusOK, gin.H{
		"event": gin.H{
			"id":            event.ID,
			"location_name": event.LocationName,
			"latitude":      event.Latitude,
			"longitude":     event.Longitude,
		},
		"journals": journals,
	})
}

func GetEvents(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			e.id,
			e.location_name,
			e.latitude,
			e.longitude,
			COUNT(j.id) AS journal_count
		FROM events e
		LEFT JOIN journals j 
			ON j.event_id = e.id 
			AND j.is_public = true
		GROUP BY e.id
		ORDER BY journal_count DESC
	`

	rows, err := config.DB.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch events"})
		return
	}
	defer rows.Close()

	var events []gin.H

	for rows.Next() {
		var id int
		var name string
		var lat, lng float64
		var count int

		if err := rows.Scan(&id, &name, &lat, &lng, &count); err != nil {
			continue
		}

		events = append(events, gin.H{
			"id":            id,
			"location_name": name,
			"latitude":      lat,
			"longitude":     lng,
			"journal_count": count,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": events,
	})
}
