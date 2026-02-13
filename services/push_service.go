package services

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
)

func SendPushToToken(token, title, body string, data map[string]string) error {

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	_, err := MessagingClient.Send(context.Background(), message)
	if err != nil {
		log.Println("Push failed:", err)
		return err
	}

	return nil
}

func BroadcastToAllUsers(title, body string, data map[string]string) error {

	message := &messaging.Message{
		Topic: "all-users",
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	_, err := MessagingClient.Send(context.Background(), message)
	if err != nil {
		log.Println("Broadcast failed:", err)
		return err
	}

	return nil
}
