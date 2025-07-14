package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"chat-poc/db"
	"chat-poc/entity"
	"chat-poc/mqtt"
	userstatus "chat-poc/user_status"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Permite qualquer origem para fins de demonstração
	},
}

// Cliente WebSocket
type WSClient struct {
	conn *websocket.Conn
	send chan []byte
}

var (
	clients       = make(map[*WSClient]bool)  // Conexões WebSocket ativas
	broadcast     = make(chan entity.Message) // Canal para enviar mensagens para todos os clientes WebSocket
	statusUpdates = make(chan map[string]bool)
)

func main() {
	// Inicializa o banco de dados
	db.InitDB("./chat.db")

	// Crie um novo ServeMux
	mux := http.NewServeMux()

	// Inicializa o cliente mqttlib
	mqtt.InitMQTTClient(mqttlibMessageReceived) // Passa a função para lidar com mensagens mqttlib

	// Rotas HTTP - AGORA REGISTRE NO SEU 'mux'
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/ws", wsHandler)
	mux.HandleFunc("/messages", getMessagesHandler)
	mux.HandleFunc("/status", getOnlineUsersHandler)

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),                                                 // Permite qualquer origem. Em prod, use a origem específica do seu frontend.
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),             // Métodos que sua API usa
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), // Cabeçalhos permitidos
	)(mux) // Aplica o middleware CORS ao seu multiplexador 'mux'

	// Inicia o goroutine para enviar mensagens para os clientes WebSocket
	go handleMessages()
	go handleStatusUpdates()

	log.Println("Server started on :8080")
	// Use o corsHandler no ListenAndServe
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}

// homeHandler renderiza a página HTML principal
func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// wsHandler lida com as conexões WebSocket
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	client := &WSClient{conn: conn, send: make(chan []byte, 256)}
	clients[client] = true
	log.Printf("New WebSocket client connected: %s", conn.RemoteAddr().String())

	// Publica que o usuário está online (assumindo um username via query param ou auth)
	username := r.URL.Query().Get("username")
	if username == "" {
		username = fmt.Sprintf("Guest-%d", time.Now().Unix()%1000)
	}
	mqtt.PublishUserStatus(username, true)
	userstatus.SetUserOnline(username) // Atualiza o status localmente também

	// Envia atualização de status para todos os clientes
	statusUpdates <- userstatus.GetOnlineUsers()

	// Envia mensagens existentes para o novo cliente
	messages, err := db.GetMessages(50) // Busca as últimas 50 mensagens
	if err != nil {
		log.Printf("Error getting historical messages: %v", err)
	} else {
		for i := len(messages) - 1; i >= 0; i-- { // Envia em ordem cronológica
			msgPayload, _ := json.Marshal(messages[i])
			client.send <- msgPayload
		}
	}

	// Inicia goroutine para escrever mensagens para este cliente
	go func() {
		defer func() {
			conn.Close()
			delete(clients, client)
			log.Printf("WebSocket client disconnected: %s", conn.RemoteAddr().String())
			mqtt.PublishUserStatus(username, false) // Publica que o usuário está offline
			userstatus.SetUserOffline(username)     // Atualiza o status localmente

			// Envia atualização de status para todos os clientes restantes
			statusUpdates <- userstatus.GetOnlineUsers()
		}()

		for {
			select {
			case message := <-client.send:
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error writing to WebSocket client %s: %v", conn.RemoteAddr().String(), err)
					return
				}
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Assume que a mensagem do cliente é um JSON com "username" e "content"
		var chatMsg struct {
			Username string `json:"username"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal(message, &chatMsg); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			continue
		}

		// Publica a mensagem no broker mqttlib
		err = mqtt.PublishMessage(chatMsg.Username, chatMsg.Content)
		if err != nil {
			log.Printf("Error publishing message to mqttlib: %v", err)
		}
	}
}

// mqttlibMessageReceived é o handler para mensagens mqttlib recebidas
func mqttlibMessageReceived(client mqttlib.Client, msg mqttlib.Message) {
	log.Printf("Received mqttlib message: %s from topic: %s", msg.Payload(), msg.Topic())

	var chatMsg entity.Message
	if err := json.Unmarshal(msg.Payload(), &chatMsg); err != nil {
		log.Printf("Error unmarshaling mqttlib message: %v", err)
		return
	}

	// Salva a mensagem no banco de dados
	err := db.SaveMessage(chatMsg.Username, chatMsg.Content)
	if err != nil {
		log.Printf("Error saving message to DB: %v", err)
	}

	// Envia a mensagem para todos os clientes WebSocket conectados
	broadcast <- chatMsg
}

// handleMessages envia mensagens do canal de broadcast para todos os clientes WebSocket
func handleMessages() {
	for {
		message := <-broadcast
		msgPayload, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshaling message for broadcast: %v", err)
			continue
		}

		for client := range clients {
			select {
			case client.send <- msgPayload:
				// Mensagem enviada com sucesso
			default:
				// Canal cheio, remove o cliente
				log.Printf("Client send channel full, removing client: %s", client.conn.RemoteAddr().String())
				client.conn.Close()
				delete(clients, client)
			}
		}
	}
}

func handleStatusUpdates() {
	for {
		usersMap := <-statusUpdates
		onlineStatusPayload := map[string]interface{}{
			"type":        "user_status",
			"onlineUsers": usersMap,
		}
		payload, err := json.Marshal(onlineStatusPayload)
		if err != nil {
			log.Printf("Error marshaling online users status for broadcast: %v", err)
			continue
		}

		for client := range clients {
			select {
			case client.send <- payload:
				// Mensagem enviada com sucesso
			default:
				// Canal cheio, remove o cliente
				log.Printf("Client send channel full, removing client: %s", client.conn.RemoteAddr().String())
				client.conn.Close()
				delete(clients, client)
			}
		}
	}
}

// getMessagesHandler retorna as últimas mensagens via HTTP
func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	messages, err := db.GetMessages(50) // Retorna as últimas 50 mensagens
	if err != nil {
		http.Error(w, "Error fetching messages", http.StatusInternalServerError)
		log.Printf("Error fetching messages: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// getOnlineUsersHandler retorna a lista de usuários online
func getOnlineUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := userstatus.GetOnlineUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
