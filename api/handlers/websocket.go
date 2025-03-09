package handlers

import (
	"log"
	"net/http"

	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Register client
	wsManager := communication.GetWSManager()
	wsManager.Register() <- conn

	// Handle disconnection
	go func() {
		<-c.Done()
		wsManager.Unregister() <- conn
	}()
}
