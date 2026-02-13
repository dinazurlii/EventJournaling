package services

import (
	"context"
	"event-journal-backend/config"
)

func SaveUserFCMToken(userID int, token string) error {

	query := `
	UPDATE event_journal.users
	SET fcm_token = $1
	WHERE id = $2
	`

	_, err := config.DB.Exec(
		context.Background(),
		query,
		token,
		userID,
	)

	return err
}
