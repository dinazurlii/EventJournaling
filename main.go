package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"event-journal-backend/config"
	"event-journal-backend/routes"
)

func main() {
	config.ConnectDB()

	r := gin.Default()

	// static files
	r.Static("/uploads", "./uploads")

	// routes API
	routes.SetupRoutes(r)

	log.Fatal(r.Run(":8080"))
}
