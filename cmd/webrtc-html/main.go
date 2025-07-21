package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Chat</title>
</head>
<body>
    <div id="messages"></div>
    <form id="messageForm">
        <input type="text" id="messageInput" required>
        <button type="submit">Send</button>
    </form>
    <script>
        const messagesDiv = document.getElementById('messages');
        const form = document.getElementById('messageForm');
        const input = document.getElementById('messageInput');

        function pollMessages() {
            fetch('/messages')
                .then(response => response.json())
                .then(message => {
                    const p = document.createElement('p');
                    p.textContent = message.text;
                    messagesDiv.appendChild(p);
                    pollMessages();
                })
                .catch(() => setTimeout(pollMessages, 1000));
        }

        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            await fetch('/send', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({text: input.value})
            });
            input.value = '';
        });

        pollMessages();
    </script>
</body>
</html>
`

type Message struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	messages    []Message
	messagesMux sync.RWMutex
	subscribers []chan Message
	subMux      sync.RWMutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	consoleURI := os.Getenv("CONSOLEURI")
	if consoleURI == "" {
		consoleURI = "127.0.0.1:2137"
	}
	wsURL := fmt.Sprintf("ws://%s/lobby", consoleURI)
	// grpcURL := fmt.Sprintf("http://%s/grpc", consoleURI)

	gameID := os.Getenv("GAMEROOM")
	if gameID == "" {
		gameID = "room"
	}

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "HOST"
	}

	p2pProxy := p2p.ProxyP2P{}

	ctx := context.Background()

	var session *bsession.Session

	if mode == "HOST" {
		session = &bsession.Session{
			RWMutex:               sync.RWMutex{},
			ID:                    "host",
			UserID:                1,
			Username:              "hostplayer",
			CharacterID:           10,
			ClassType:             0,
			Conn:                  nil,
			OnceSelectedCharacter: sync.Once{},
			State:                 nil,
		}
	} else if mode == "JOIN" {
		session = &bsession.Session{
			RWMutex:               sync.RWMutex{},
			ID:                    "guest1",
			UserID:                2,
			Username:              "joiner1",
			CharacterID:           20,
			ClassType:             0,
			Conn:                  nil,
			OnceSelectedCharacter: sync.Once{},
			State:                 nil,
		}
	}

	if err := session.ConnectOverWebsocket(ctx, &v1.User{UserId: session.UserID, Username: session.Username}, wsURL); err != nil {
		log.Fatal(err)
	}

	if err := session.JoinLobby(ctx); err != nil {
		log.Fatal("failed to join lobby over websocket", logging.Error(err))
	}

	px := p2pProxy.Create(session)

	handlers := []proxy.MessageHandler{
		// backend.NewLobbyEventHandler(session),
		px.Handle,
	}
	observe := func(ctx context.Context, wsConn *websocket.Conn) {
		for {
			if ctx.Err() != nil {
				return
			}

			// Read the broadcast and handle them as commands.
			p, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", session.ID, logging.Error(err))
				return
			}

			// slog.Debug("Signal from lobby", "type", et.String(), "session", session.ID, "payload", string(p[1:]))

			// TODO: Register handlers and handle them here.
			for _, handle := range handlers {
				if err := handle(ctx, p); err != nil {
					slog.Error("Error handling message", "session", session.ID, logging.Error(err))
					return
				}
			}
		}
	}
	if err := session.StartObserver(ctx, observe); err != nil {
		log.Fatal(err)
	}

	if mode == "HOST" {
		roomIP, err := px.CreateRoom(ctx, proxy.CreateParams{GameID: gameID})
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Created room:", roomIP)

		if err := px.HostRoom(ctx, proxy.HostParams{GameID: gameID}); err != nil {
			log.Fatal(err)
		}
	} else if mode == "JOIN" {
		// gm := multiv1connect.NewGameServiceClient(http.DefaultClient, grpcURL)

		// roomIP, err := p2pProxy.CreateRoom(proxy.CreateParams{GameID: gameID}, session)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// log.Println("Created room:", roomIP)
		//
		// if err := p2pProxy.HostRoom(ctx, proxy.HostParams{GameID: gameID}, session); err != nil {
		// 	log.Fatal(err)
		// }
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("chat").Parse(htmlTemplate))
		tmpl.Execute(w, nil)
	})

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		msg.Timestamp = time.Now()

		messagesMux.Lock()
		messages = append(messages, msg)
		messagesMux.Unlock()

		subMux.RLock()
		for _, ch := range subscribers {
			ch <- msg
		}
		subMux.RUnlock()
	})

	http.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		messageChan := make(chan Message)

		subMux.Lock()
		subscribers = append(subscribers, messageChan)
		subMux.Unlock()

		defer func() {
			subMux.Lock()
			for i, ch := range subscribers {
				if ch == messageChan {
					subscribers = append(subscribers[:i], subscribers[i+1:]...)
					break
				}
			}
			subMux.Unlock()
		}()

		select {
		case msg := <-messageChan:
			json.NewEncoder(w).Encode(msg)
		case <-time.After(30 * time.Second):
			w.WriteHeader(http.StatusNoContent)
		}
	})

	http.ListenAndServe(net.JoinHostPort("", port), nil)
}
