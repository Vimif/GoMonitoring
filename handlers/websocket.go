package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"go-monitoring/models"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 30 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// Si pas d'origin (requête directe), accepter
		if origin == "" {
			return true
		}

		// Liste blanche des origines autorisées
		allowedOrigins := []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			"https://localhost:8080",
			"https://127.0.0.1:8080",
		}

		// Vérifier si l'origine est dans la liste blanche
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}

		// Log des tentatives d'accès non autorisées
		log.Printf("WebSocket: origine non autorisée refusée: %s", origin)
		return false
	},
}

// Global Hub instance
var WSHub = NewHub()

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []models.Machine
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []models.Machine),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("Client WS connecté")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Println("Client WS déconnecté")

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				client.SetWriteDeadline(time.Now().Add(writeWait))
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("WS Error: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast envoie les mises à jour
func (h *Hub) Broadcast(machines []models.Machine) {
	h.broadcast <- machines
}

// ServeWS gère la connexion WebSocket
func ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS Upgrade Error:", err)
		return
	}

	WSHub.register <- conn

	// Configure pong handler
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Ping ticker to keep connection alive
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		defer func() {
			WSHub.unregister <- conn
		}()

		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read loop to handle close and pong responses
	go func() {
		defer func() {
			WSHub.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
