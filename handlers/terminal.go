package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// TerminalMessage définit le protocole de communication
type TerminalMessage struct {
	Type string `json:"type"` // "input", "resize"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// WebTerminalHandler gère la connexion WebSocket pour le terminal
func WebTerminalHandler(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID requis", http.StatusBadRequest)
			return
		}

		// Upgrade en WebSocket
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Terminal: Erreur upgrade WS: %v", err)
			return
		}
		defer ws.Close()

		// Récupérer le client SSH
		pool := cm.GetPool()
		sshClientWrapper, err := pool.GetClient(id)
		if err != nil {
			ws.WriteMessage(websocket.TextMessage, []byte("Erreur: Machine introuvable\r\n"))
			return
		}

		// Créer la session SSH
		session, err := sshClientWrapper.NewSession()
		if err != nil {
			ws.WriteMessage(websocket.TextMessage, []byte("Erreur: Impossible de créer la session SSH: "+err.Error()+"\r\n"))
			return
		}
		defer session.Close()

		// Pipes
		stdin, err := session.StdinPipe()
		if err != nil {
			return
		}
		stdout, err := session.StdoutPipe()
		if err != nil {
			return
		}
		stderr, err := session.StderrPipe()
		if err != nil {
			return
		}

		// Request PTY
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		// Default size, will be resized by client
		if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
			ws.WriteMessage(websocket.TextMessage, []byte("Erreur: RequestPty failed\r\n"))
			return
		}

		// Start shell
		if err := session.Shell(); err != nil {
			ws.WriteMessage(websocket.TextMessage, []byte("Erreur: Shell failed\r\n"))
			return
		}

		// Channel pour synchronisation
		done := make(chan struct{})
		var wg sync.WaitGroup

		// Goroutine: SSH Stdout -> WS
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Terminal: Stdout read error: %v", err)
					}
					break
				}
				// Envoyer les données brutes
				if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					log.Printf("Terminal: WS Write error: %v", err)
					break
				}
			}
			close(done)
		}()

		// Goroutine: SSH Stderr -> WS (mixed)
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := stderr.Read(buf)
				if err != nil {
					break
				}
				ws.WriteMessage(websocket.TextMessage, buf[:n])
			}
		}()

		// Loop: WS -> SSH Stdin (Main Loop)
		// Send ping
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
				}
			}
		}()

		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}

			// Parse JSON message
			var termMsg TerminalMessage
			if err := json.Unmarshal(msg, &termMsg); err != nil {
				// Fallback: treat as raw input if not JSON? No, strictly JSON from our client
				continue
			}

			switch termMsg.Type {
			case "input":
				stdin.Write([]byte(termMsg.Data))
			case "resize":
				session.WindowChange(termMsg.Rows, termMsg.Cols)
			}
		}

		wg.Wait()
	}
}
