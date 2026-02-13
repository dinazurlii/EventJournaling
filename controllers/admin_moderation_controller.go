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

//
// ===== HELPER =====
//

func getEventData(ctx context.Context, eventID int) (string, string, string, int, error) {
	var title, email, fcmToken string
	var userID int

	query := `
	SELECT e.title, u.email, u.fcm_token, u.id
	FROM event_journal.events e
	JOIN event_journal.users u ON u.id = e.created_by
	WHERE e.id = $1
	`

	err := config.DB.QueryRow(ctx, query, eventID).
		Scan(&title, &email, &fcmToken, &userID)

	return title, email, fcmToken, userID, err
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
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

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

	CreateModerationLog(context.Background(), eventID, adminID, "approved", nil)

	// Get event data (creator info)
	title, email, fcmToken, userID, err := getEventData(context.Background(), eventID)
	if err == nil {

		// Email
		go services.SendEventApproved(email, title, eventID)

		// Push to creator
		if fcmToken != "" {
			go services.SendPushToToken(
				fcmToken,
				"Event Approved 🎉",
				"Your event '"+title+"' has been approved!",
				map[string]string{
					"type":     "event_approved",
					"event_id": strconv.Itoa(eventID),
				},
			)
		}

		// Save notification history
		go services.SaveNotification(
			userID,
			"Event Approved 🎉",
			"Your event '"+title+"' has been approved!",
		)
	}

	// Broadcast to all users
	go services.BroadcastToAllUsers(
		"New Event Available 🎊",
		title,
		map[string]string{
			"type":     "new_event",
			"event_id": strconv.Itoa(eventID),
		},
	)

	c.JSON(http.StatusOK, gin.H{"message": "event approved"})
}

//
// ===== REJECT EVENT =====
//

type RejectEventInput struct {
	Reason string `json:"reason"`
}

func RejectEvent(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

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

	CreateModerationLog(
		context.Background(),
		eventID,
		adminID,
		"rejected",
		&input.Reason,
	)

	title, email, fcmToken, userID, err := getEventData(context.Background(), eventID)
	if err == nil {

		go services.SendEventRejected(email, title, input.Reason, eventID)

		if fcmToken != "" {
			go services.SendPushToToken(
				fcmToken,
				"Event Rejected ❌",
				"Your event '"+title+"' was rejected.",
				map[string]string{
					"type":     "event_rejected",
					"event_id": strconv.Itoa(eventID),
				},
			)
		}

		go services.SaveNotification(
			userID,
			"Event Rejected ❌",
			"Your event '"+title+"' was rejected.",
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "event rejected"})
}

//
// ===== MODERATION LOG =====
//

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

		logItem := gin.H{
			"action": action,
			"admin":  adminEmail,
			"time":   createdAt,
		}

		if reason.Valid {
			logItem["reason"] = reason.String
		}

		logs = append(logs, logItem)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
