package controllers

import (
	"context"
	"net/http"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

// ===== INPUT STRUCT =====

type CreateEventInput struct {
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	EventDate       time.Time `json:"event_date"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	LocationName    string    `json:"location_name"`
	IsPaid          bool      `json:"is_paid"`
	RegistrationURL string    `json:"registration_url"`
}

type UpdateEventInput struct {
	Title           *string    `json:"title"`
	Description     *string    `json:"description"`
	EventDate       *time.Time `json:"event_date"`
	Latitude        *float64   `json:"latitude"`
	Longitude       *float64   `json:"longitude"`
	LocationName    *string    `json:"location_name"`
	IsPaid          *bool      `json:"is_paid"`
	RegistrationURL *string    `json:"registration_url"`
}

// ===== CREATE EVENT =====

func CreateEvent(c *gin.Context) {
	var input CreateEventInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// business rule
	if input.IsPaid && input.RegistrationURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "registration_url wajib untuk event berbayar",
		})
		return
	}

	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
	INSERT INTO events (
		title, description, event_date,
		latitude, longitude, location_name,
		is_paid, registration_url,
		status, created_by
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',$9)
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
		input.IsPaid,
		input.RegistrationURL,
		userID,
	).Scan(&eventID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     eventID,
		"status": "pending",
	})
}

//
// ===== GET MY EVENTS =====
//

func GetMyEvents(c *gin.Context) {
	userID, _ := c.Get("user_id")

	query := `
	SELECT
	id,
	title,
	event_date,
	status,
	rejection_reason
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
		var title, location, status string
		var date time.Time
		var lat, lng float64

		if err := rows.Scan(&id, &title, &date, &lat, &lng, &location, &status); err != nil {
			continue
		}

		events = append(events, gin.H{
			"id":        id,
			"title":     title,
			"date":      date,
			"latitude":  lat,
			"longitude": lng,
			"location":  location,
			"status":    status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": events})
}

//
// ===== GET PUBLIC EVENTS =====
//

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
	WHERE e.status = 'approved'
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

	c.JSON(http.StatusOK, gin.H{"data": events})
}

//
// ===== GET EVENT DETAIL (PUBLIC) =====
//

func GetEventDetail(c *gin.Context) {
	eventID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var event struct {
		ID           int
		Title        string
		EventDate    time.Time
		Latitude     float64
		Longitude    float64
		LocationName string
	}

	eventQuery := `
	SELECT id, title, event_date, latitude, longitude, location_name
	FROM events
	WHERE id = $1 AND status = 'approved'
	`

	err := config.DB.QueryRow(ctx, eventQuery, eventID).
		Scan(&event.ID, &event.Title, &event.EventDate, &event.Latitude, &event.Longitude, &event.LocationName)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	journalQuery := `
	SELECT id, title, content, created_at
	FROM journals
	WHERE event_id = $1 AND is_public = true
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

	c.JSON(http.StatusOK, gin.H{
		"event":    event,
		"journals": journals,
	})
}

//
// ===== UPDATE EVENT =====
//

func UpdateEvent(c *gin.Context) {
	eventID := c.Param("id")

	var input UpdateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var createdBy int
	var isPaid bool
	var registrationURL *string

	err := config.DB.QueryRow(
		ctx,
		`SELECT created_by, is_paid, registration_url FROM events WHERE id = $1`,
		eventID,
	).Scan(&createdBy, &isPaid, &registrationURL)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	if role != "admin" && createdBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}

	finalIsPaid := isPaid
	if input.IsPaid != nil {
		finalIsPaid = *input.IsPaid
	}

	finalRegistrationURL := registrationURL
	if input.RegistrationURL != nil {
		finalRegistrationURL = input.RegistrationURL
	}

	if finalIsPaid && (finalRegistrationURL == nil || *finalRegistrationURL == "") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "registration_url wajib untuk event berbayar",
		})
		return
	}

	status := "pending"
	if role == "admin" {
		status = "approved"
	}

	query := `
	UPDATE events SET
		title = COALESCE($1, title),
		description = COALESCE($2, description),
		event_date = COALESCE($3, event_date),
		latitude = COALESCE($4, latitude),
		longitude = COALESCE($5, longitude),
		location_name = COALESCE($6, location_name),
		is_paid = COALESCE($7, is_paid),
		registration_url = COALESCE($8, registration_url),
		status = $9
	WHERE id = $10
	`

	_, err = config.DB.Exec(
		ctx,
		query,
		input.Title,
		input.Description,
		input.EventDate,
		input.Latitude,
		input.Longitude,
		input.LocationName,
		input.IsPaid,
		input.RegistrationURL,
		status,
		eventID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "event updated"})
}
