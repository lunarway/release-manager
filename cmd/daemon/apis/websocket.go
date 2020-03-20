package apis

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// Handle Flux WebSocket connections
func HandleWebsocket(config APIConfig) error {
	log := config.Log.With("subtype", "websocket")
	config.Server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.With("URL", r.URL).Info("Request for URL")
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.With("error", err).Info("websocket upgrade error")
			return
		}
		defer func() {
			log.Info("client disconnected")
			c.Close()
		}()

		log.Info("client connected")

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.With("error", err).Error("read error")
				break
			}

			log.With("message", message).Info("recieved message")
			err = c.WriteMessage(mt, message)

			if err != nil {
				log.With("error", err).Error("websocket write error", err)
				break
			}
		}
	})

	return nil
}
