package handlers

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/srgchrksv/geminipodcaster/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartPodcast(c *gin.Context, services services.Services) {
	session := sessions.Default(c)
	sessionID := session.Get("sessionID")
	if sessionID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No sessions found"})
		return
	}
	log.Println("Session ID:", sessionID)
	// get podcastContext from the request
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	podcastSession := sessionID.(string)

	services.Podcast(c, conn, podcastSession)

}

// route for users interaction
func PodcastInteraction(c *gin.Context, services services.Services) {
	session := sessions.Default(c)
	sessionID := session.Get("sessionID")
	if sessionID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No sessions found"})
		return
	}

	err := services.UserInteraction(c, sessionID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Users interaction."})

}
