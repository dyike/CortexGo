package main

import (
	"log"
	"net/http"

	cortexgo "github.com/dyike/CortexGo/cortex_go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development, should be restricted in production
	},
}

func generateResponse(task string, ws *websocket.Conn) ([]string, error) {
	instructor := cortexgo.NewSystemInstructor()
	if ws != nil {
		instructor.SetWebSocket(ws)
	}

	response, err := instructor.Run(task)
	if err != nil {
		return nil, err
	}

	// Cleanup
	if err := instructor.Shutdown(); err != nil {
		log.Printf("Warning: shutdown error: %v", err)
	}

	return response, nil
}

func main() {
	r := gin.Default()

	// HTTP endpoint
	r.GET("/agent/chat", func(c *gin.Context) {
		task := c.Query("task")
		if task == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task parameter is required"})
			return
		}

		response, err := generateResponse(task, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}
		defer ws.Close()

		for {
			// Read message
			messageType, message, err := ws.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				break
			}

			if messageType != websocket.TextMessage {
				continue
			}

			// Process message
			_, err = generateResponse(string(message), ws)
			if err != nil {
				log.Printf("Error generating response: %v", err)
				if err := ws.WriteMessage(websocket.TextMessage, []byte(err.Error())); err != nil {
					log.Printf("Error sending error message: %v", err)
				}
				break
			}
		}
	})

	log.Printf("Server starting on :8080")
	log.Fatal(r.Run(":8080"))
}
