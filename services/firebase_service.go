package services

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var MessagingClient *messaging.Client

func InitFirebase() {

	ctx := context.Background()

	opt := option.WithCredentialsFile("serviceAccountKey.json") // <-- TARUH FILE DI ROOT PROJECT

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("Firebase init error:", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Fatal("Firebase messaging error:", err)
	}

	MessagingClient = client

	log.Println("🔥 Firebase initialized")
}
