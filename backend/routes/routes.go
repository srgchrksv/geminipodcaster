package routes

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/srgchrksv/geminipodcaster/handlers"
	"github.com/srgchrksv/geminipodcaster/models"
	"github.com/srgchrksv/geminipodcaster/services"
)

func RegisterRoutes(r *gin.Engine, model models.Gemini, services *services.Services) {
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))
	// Configure CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"} // Change this to your frontend URL
	config.AllowMethods = []string{"GET", "POST"}
	config.AllowHeaders = []string{"Content-Type", "text/plain", "application/json"} // Combine all necessary headers
	config.AllowCredentials = true                                                   // Allow credentials (cookies)
	r.Use(cors.New(config))

	r.GET("/", func(c *gin.Context) {
		// Create a new session ID
		session := sessions.Default(c)
		if session.Get("sessionID") == nil {
			session.Set("sessionID", uuid.New().String())
			session.Save()
		}
		podcastSession := session.Get("sessionID").(string)
		c.JSON(http.StatusOK, gin.H{"message": "Podcast starting...", "session_id": podcastSession})
	})
	r.GET("/podcast", func(c *gin.Context) {
		handlers.StartPodcast(c, model, *services)
	})
	r.POST("/interact", func(c *gin.Context) {
		handlers.PodcastInteraction(c, *services)
	})
}
