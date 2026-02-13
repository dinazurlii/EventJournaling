package services

import (
	"context"
	"event-journal-backend/config"
)

func SaveNotification(userID int, title, body string) error {

	query := `
	INSERT INTO event_journal.notifications
		(user_id, title, body)
	VALUES ($1, $2, $3)
	`

	_, err := config.DB.Exec(
		context.Background(),
		query,
		userID,
		title,
		body,
	)

	return err
}
