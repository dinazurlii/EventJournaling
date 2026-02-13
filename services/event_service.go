package services

import (
	"context"
	"event-journal-backend/config"
)

func GetEventData(ctx context.Context, eventID string) (string, string, string, int, error) {
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
