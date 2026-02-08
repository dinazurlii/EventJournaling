package controllers

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"event-journal-backend/config"

	"github.com/gin-gonic/gin"
)

type CreateJournalInput struct {
	EventID   int     `json:"event_id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	IsPublic  bool    `json:"-"`
}

func CreateJournal(c *gin.Context) {
	var input struct {
		EventID   *int    `json:"event_id"` // pointer → optional
		Title     string  `json:"title" binding:"required"`
		Content   string  `json:"content"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		IsPublic  bool    `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt("user_id")

	// 🔐 public journal wajib punya lokasi
	if input.IsPublic && (input.Latitude == 0 || input.Longitude == 0) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "public journal must have location",
		})
		return
	}

	// 🔎 VALIDASI EVENT (JIKA ADA)
	if input.EventID != nil {
		var exists bool
		err := config.DB.QueryRow(
			context.Background(),
			`
			SELECT EXISTS (
				SELECT 1 FROM events
				WHERE id = $1 AND event_type = 'organizer'
			)
			`,
			*input.EventID,
		).Scan(&exists)

		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid event_id",
			})
			return
		}
	}

	query := `
		INSERT INTO journals (
			user_id, event_id, title, content,
			latitude, longitude, is_public
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`

	_, err := config.DB.Exec(
		context.Background(),
		query,
		userID,
		input.EventID,
		input.Title,
		input.Content,
		input.Latitude,
		input.Longitude,
		input.IsPublic,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "journal created",
	})
}

func GetMyJournals(c *gin.Context) {
	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, title, content, created_at
		FROM journals
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := config.DB.Query(ctx, query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch journals"})
		return
	}
	defer rows.Close()

	var journals []gin.H

	for rows.Next() {
		var id int
		var title, content string
		var createdAt time.Time

		err := rows.Scan(&id, &title, &content, &createdAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read journal"})
			return
		}

		journals = append(journals, gin.H{
			"id":         id,
			"title":      title,
			"content":    content,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": journals,
	})
}
func GetPublicJournals(c *gin.Context) {
	lat := c.Query("lat")
	lng := c.Query("lng")
	radius := c.DefaultQuery("radius", "5")

	query := `
		SELECT id, title, content, latitude, longitude, created_at
		FROM journals
		WHERE is_public = true
		  AND latitude IS NOT NULL
		  AND longitude IS NOT NULL
		  AND (
		    6371 * acos(
		      cos(radians($1)) *
		      cos(radians(latitude)) *
		      cos(radians(longitude) - radians($2)) +
		      sin(radians($1)) *
		      sin(radians(latitude))
		    )
		  ) <= $3
		ORDER BY created_at DESC
	`

	rows, err := config.DB.Query(
		context.Background(),
		query,
		lat, lng, radius,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch public journals"})
		return
	}
	defer rows.Close()

	var journals []gin.H

	for rows.Next() {
		var id int
		var title, content string
		var lat, lng float64
		var createdAt time.Time

		rows.Scan(&id, &title, &content, &lat, &lng, &createdAt)

		journals = append(journals, gin.H{
			"id":         id,
			"title":      title,
			"content":    content,
			"latitude":   lat,
			"longitude":  lng,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": journals,
	})
}
func GetJournalDetail(c *gin.Context) {
	journalID := c.Param("id")

	bookmarked := false
	// optional: user login atau enggak
	userID, _ := c.Get("user_id")

	query := `
		SELECT 
			j.id,
			j.title,
			j.content,
			j.latitude,
			j.longitude,
			j.is_public,
			j.created_at,
			u.id AS author_id,
			u.email
		FROM journals j
		JOIN users u ON u.id = j.user_id
		WHERE j.id = $1
	`

	var (
		id        int
		title     string
		content   string
		lat       float64
		lng       float64
		isPublic  bool
		createdAt time.Time
		authorID  int
		email     string
	)

	err := config.DB.QueryRow(
		context.Background(),
		query,
		journalID,
	).Scan(
		&id,
		&title,
		&content,
		&lat,
		&lng,
		&isPublic,
		&createdAt,
		&authorID,
		&email,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "journal not found"})
		return
	}

	// 🔒 PRIVATE JOURNAL CHECK
	if !isPublic {
		if userID == nil || userID.(int) != authorID {
			c.JSON(http.StatusForbidden, gin.H{"error": "this journal is private"})
			return
		}
	}
	if userID != nil {
		bookmarkQuery := `
		SELECT 1 FROM bookmarks
		WHERE user_id = $1 AND journal_id = $2
	`
		err := config.DB.QueryRow(
			context.Background(),
			bookmarkQuery,
			userID,
			journalID,
		).Scan(new(int))

		bookmarked = err == nil
	}
	commentQuery := `
	SELECT c.id, c.content, c.created_at, u.id, u.email
	FROM comments c
	JOIN users u ON u.id = c.user_id
	WHERE c.journal_id = $1
	ORDER BY c.created_at ASC
`

	rows, err := config.DB.Query(
		context.Background(),
		commentQuery,
		journalID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}
	defer rows.Close()

	var comments []gin.H

	for rows.Next() {
		var (
			commentID int
			comment   string
			created   time.Time
			uid       int
			email     string
		)

		rows.Scan(&commentID, &comment, &created, &uid, &email)

		comments = append(comments, gin.H{
			"id":         commentID,
			"content":    comment,
			"created_at": created,
			"user": gin.H{
				"id":    uid,
				"email": email,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"title":      title,
		"content":    content,
		"latitude":   lat,
		"longitude":  lng,
		"is_public":  isPublic,
		"created_at": createdAt,
		"author": gin.H{
			"id":    authorID,
			"email": email,
		},
		"bookmarked": bookmarked,
		"comments":   comments,
	})
}

func GetEventJournals(c *gin.Context) {
	eventID := c.Param("id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔢 total data
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM journals
		WHERE event_id = $1
		  AND is_public = true
	`
	_ = config.DB.QueryRow(ctx, countQuery, eventID).Scan(&total)

	// 📄 data pagination
	query := `
		SELECT id, title, content, created_at
		FROM journals
		WHERE event_id = $1
		  AND is_public = true
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := config.DB.Query(ctx, query, eventID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch journals"})
		return
	}
	defer rows.Close()

	var journals []gin.H

	for rows.Next() {
		var id int
		var title, content string
		var createdAt time.Time

		if err := rows.Scan(&id, &title, &content, &createdAt); err != nil {
			continue
		}

		journals = append(journals, gin.H{
			"id":         id,
			"title":      title,
			"content":    content,
			"created_at": createdAt,
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"data": journals,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}
