package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"event-journal-backend/config"
	"event-journal-backend/routes"
	"event-journal-backend/services"
)

func main() {
	godotenv.Load()
	config.ConnectDB()
	services.InitFirebase()

	r := gin.Default()

	r.Static("/uploads", "./uploads")
	routes.SetupRoutes(r)

	log.Fatal(r.Run(":8080"))
}
