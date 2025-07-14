package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"chat-poc/db"
	"chat-poc/entity"
	"chat-poc/mqtt"
	userstatus "chat-poc/user_status"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/handlers"
)

func main() {
	// Inicializa o banco de dados
	db.InitDB("./chat.db")

	// Crie um novo ServeMux
	mux := http.NewServeMux()

	// Inicializa o cliente mqttlib
	mqtt.InitMQTTClient(mqttlibMessageReceived) // Passa a função para lidar com mensagens mqttlib

	// Rotas HTTP
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/messages", getMessagesHandler)
	mux.HandleFunc("/status", getOnlineUsersHandler)
	mux.HandleFunc("/user-activity", getUserActivityHandler)

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
	)(mux)

	log.Println("Server started on :8080")
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

// mqttlibMessageReceived é o handler para mensagens mqttlib recebidas
func mqttlibMessageReceived(client mqttlib.Client, msg mqttlib.Message) {
	log.Printf("Received mqttlib message: %s from topic: %s", msg.Payload(), msg.Topic())

	if msg.Topic() == "chat/messages" {
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
	} else if msg.Topic() == "chat/status" {
		// Deixa o userstatus.StatusHandler lidar com isso
		userstatus.StatusHandler(client, msg)
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

// getUserActivityHandler retorna o histórico de atividades de usuário
func getUserActivityHandler(w http.ResponseWriter, r *http.Request) {
	activities, err := db.GetUserActivities(100)
	if err != nil {
		http.Error(w, "Error fetching user activity", http.StatusInternalServerError)
		log.Printf("Error fetching user activity: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}
