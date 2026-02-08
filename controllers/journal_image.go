package controllers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

func UploadJournalImage(c *gin.Context) {
	journalID := c.Param("id")

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image is required"})
		return
	}

	// generate filename
	filename := fmt.Sprintf(
		"journal_%s_%d%s",
		journalID,
		time.Now().Unix(),
		filepath.Ext(file.Filename),
	)

	savePath := "uploads/journals/" + filename
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image"})
		return
	}

	imageURL := "/uploads/journals/" + filename

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO journal_images (journal_id, image_url)
		VALUES ($1, $2)
	`
	_, err = config.DB.Exec(ctx, query, journalID, imageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image record"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"image_url": imageURL,
	})
}
