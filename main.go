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

	r.Static("/uploads", "./uploads")
	routes.SetupRoutes(r)

	log.Fatal(r.Run(":8080"))
}
