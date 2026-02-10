package controllers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"event-journal-backend/config"
	"event-journal-backend/services"

	"github.com/gin-gonic/gin"
)

//
// ===== HELPER =====
//

func getEventTitleAndEmail(ctx context.Context, eventID string) (string, string, error) {
	var title, email string

	query := `
	SELECT e.title, u.email
	FROM events e
	JOIN users u ON u.id = e.created_by
	WHERE e.id = $1
	`

	err := config.DB.QueryRow(ctx, query, eventID).Scan(&title, &email)
	return title, email, err
}

//
// ===== GET PENDING EVENTS =====
//

func GetPendingEvents(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}

	query := `
	SELECT
	e.id,
	e.title,
	e.event_date,
	e.location_name,
	u.id,
	u.email,
	e.created_at
	FROM event_journal.events e
	JOIN event_journal.users u ON u.id = e.created_by
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
			eventDate    sql.NullTime
			locationName sql.NullString
			userID       sql.NullInt32
			email        sql.NullString
			createdAt    time.Time
		)

		if err := rows.Scan(
			&id,
			&title,
			&eventDate,
			&locationName,
			&userID,
			&email,
			&createdAt,
		); err != nil {
			continue
		}

		events = append(events, gin.H{
			"id":         id,
			"title":      title,
			"event_date": eventDate,
			"location":   locationName,
			"created_at": createdAt,
			"creator": gin.H{
				"id":    userID,
				"email": email,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": events})
}

//
// ===== APPROVE EVENT =====
//

func ApproveEvent(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}

	eventID := c.Param("id")

	query := `
	UPDATE events
	SET status = 'approved'
	WHERE id = $1 AND status = 'pending'
	`

	result, err := config.DB.Exec(context.Background(), query, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found or already processed",
		})
		return
	}

	// ✉️ SEND EMAIL (TEMPLATE)
	title, email, err := getEventTitleAndEmail(context.Background(), eventID)
	if err == nil {
		body, err := services.RenderEmailTemplate(
			"event_approved.html",
			gin.H{
				"Title": title,
			},
		)

		if err == nil {
			go services.SendEmail(
				email,
				"🎉 Event kamu disetujui",
				body,
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "event approved"})
}

//
// ===== REJECT EVENT =====
//

type RejectEventInput struct {
	Reason string `json:"reason"`
}

func RejectEvent(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}

	eventID := c.Param("id")

	var input RejectEventInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "rejection reason is required",
		})
		return
	}

	query := `
	UPDATE events
	SET status = 'rejected',
		rejection_reason = $2,
		rejected_at = NOW()
	WHERE id = $1 AND status = 'pending'
	`

	result, err := config.DB.Exec(
		context.Background(),
		query,
		eventID,
		input.Reason,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found or already processed",
		})
		return
	}

	// ✉️ SEND EMAIL (TEMPLATE)
	title, email, err := getEventTitleAndEmail(context.Background(), eventID)
	if err == nil {
		body, err := services.RenderEmailTemplate(
			"event_rejected.html",
			gin.H{
				"Title":  title,
				"Reason": input.Reason,
			},
		)

		if err == nil {
			go services.SendEmail(
				email,
				"❌ Event kamu ditolak",
				body,
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "event rejected"})
}
