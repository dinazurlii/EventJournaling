package routes

import (
	"event-journal-backend/controllers"
	"event-journal-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	{
		api.POST("/register", controllers.Register)
		api.POST("/login", controllers.Login)

		// PROTECTED ROUTES
		api.GET("/me", middleware.JWTAuthMiddleware(), controllers.Me)
		api.POST("/bookmarks", middleware.JWTAuthMiddleware(), controllers.BookmarkJournal)
		api.DELETE("/bookmarks/:journal_id", middleware.JWTAuthMiddleware(), controllers.UnbookmarkJournal)
		api.GET("/bookmarks", middleware.JWTAuthMiddleware(), controllers.GetMyBookmarks)

		// LEGACY EVENTS (USER / MARKER ONLY)
		api.POST("/events", middleware.JWTAuthMiddleware(), controllers.CreateEvent)
		api.GET("/events", middleware.JWTAuthMiddleware(), controllers.GetMyEvents)

		// ⬇️ HARUS DI ATAS :id
		api.GET("/events/all", controllers.GetEvents)
		api.GET("/events/:id/journals", controllers.GetEventJournals)

		// ⬇️ PALING BAWAH
		api.GET("/events/:id", controllers.GetEventDetail)

		api.POST("/journals", middleware.JWTAuthMiddleware(), controllers.CreateJournal)
		api.GET("/journals", middleware.JWTAuthMiddleware(), controllers.GetMyJournals)
		api.GET("/journals/public", controllers.GetPublicJournals)

		// MAP ROUTES
		api.GET("/map/journals", controllers.GetMapJournals)

		// JOURNAL LIKES ROUTES
		api.POST("/journals/:id/like",
			middleware.JWTAuthMiddleware(),
			controllers.ToggleJournalLike,
		)

		api.GET("/journals/:id/likes",
			controllers.GetJournalLikes,
		)

		// UPLOAD ROUTES
		api.POST(
			"/journals/:id/images",
			middleware.JWTAuthMiddleware(),
			controllers.UploadJournalImage,
		)

		// COMMENT ROUTES
		api.POST("/journals/:id/comments", middleware.JWTAuthMiddleware(), controllers.CreateComment)
		api.GET("/journals/:id/comments", controllers.GetJournalComments)
		api.DELETE("/comments/:id", middleware.JWTAuthMiddleware(), controllers.DeleteComment)

		//BOOKMARK ROUTES
		api.GET("/journals/:id", middleware.OptionalJWT(), controllers.GetJournalDetail)

		// ORGANIZER EVENTS (PUBLIC ACTIVITIES)
		api.POST("/organizer/events",
			middleware.JWTAuthMiddleware(),
			controllers.CreateOrganizerEvent,
		)

		api.GET("/organizer/events",
			controllers.SearchOrganizerEvents,
		)

		api.GET("/organizer/events/:id",
			controllers.GetOrganizerEventDetail,
		)
		// ADMIN ROUTES
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuthMiddleware(), middleware.AdminOnly())
		{
			admin.GET("/events/pending", controllers.GetPendingEvents)
			admin.PUT("/events/:id/approve", controllers.ApproveEvent)
			admin.PUT("/events/:id/reject", controllers.RejectEvent)
		}
	}

}
