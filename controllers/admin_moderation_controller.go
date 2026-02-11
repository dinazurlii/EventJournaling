package controllers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"event-journal-backend/config"
	"event-journal-backend/services"

	"github.com/gin-gonic/gin"
)

// ===== HELPER =====

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

// ===== GET PENDING EVENTS =====

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
	eventID, _ := strconv.Atoi(c.Param("id"))
	adminID := c.GetInt("user_id")

	query := `
		UPDATE event_journal.events
		SET status = 'approved',
		    rejection_reason = NULL,
		    rejected_at = NULL
		WHERE id = $1 AND status = 'pending'
	`

	result, err := config.DB.Exec(context.Background(), query, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve event"})
		return
	}

	rows := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found or already processed"})
		return
	}

	CreateModerationLog(
		context.Background(),
		eventID,
		adminID,
		"approved",
		nil,
	)

	title, email, err := getEventTitleAndEmail(context.Background(), strconv.Itoa(eventID))
	if err == nil {
		services.SendEventApproved(email, title)
	}

	c.JSON(http.StatusOK, gin.H{"message": "event approved"})
}

// ===== REJECT EVENT =====

type RejectEventInput struct {
	Reason string `json:"reason"`
}

func RejectEvent(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("id"))
	adminID := c.GetInt("user_id")

	var input RejectEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rejection reason required"})
		return
	}

	query := `
		UPDATE event_journal.events
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject event"})
		return
	}

	rows := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found or already processed"})
		return
	}

	//  INSERT LOG
	CreateModerationLog(
		context.Background(),
		eventID,
		adminID,
		"rejected",
		&input.Reason,
	)

	title, email, err := getEventTitleAndEmail(context.Background(), strconv.Itoa(eventID))
	if err == nil {
		services.SendEventRejected(email, title, input.Reason)
	}

	c.JSON(http.StatusOK, gin.H{"message": "event rejected"})
}

func CreateModerationLog(
	ctx context.Context,
	eventID int,
	adminID int,
	action string,
	reason *string,
) {

	query := `
		INSERT INTO event_journal.event_moderation_logs
			(event_id, admin_id, action, reason)
		VALUES ($1, $2, $3, $4)
	`

	_, err := config.DB.Exec(
		ctx,
		query,
		eventID,
		adminID,
		action,
		reason,
	)

	if err != nil {
		log.Println("FAILED TO INSERT MODERATION LOG:", err)
	}
}

func GetEventModerationLogs(c *gin.Context) {
	eventID := c.Param("id")

	query := `
		SELECT
			l.action,
			l.reason,
			l.created_at,
			u.email
		FROM event_journal.event_moderation_logs l
		JOIN event_journal.users u ON u.id = l.admin_id
		WHERE l.event_id = $1
		ORDER BY l.created_at DESC
	`

	rows, err := config.DB.Query(context.Background(), query, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch logs"})
		return
	}
	defer rows.Close()

	var logs []gin.H

	for rows.Next() {
		var action string
		var reason sql.NullString
		var createdAt time.Time
		var adminEmail string

		rows.Scan(&action, &reason, &createdAt, &adminEmail)

		log := gin.H{
			"action": action,
			"admin":  adminEmail,
			"time":   createdAt,
		}

		if reason.Valid {
			log["reason"] = reason.String
		}

		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
