package userstatus

import (
	"encoding/json"
	"log"
	"sync"

	"chat-poc/db"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	onlineUsers = make(map[string]bool)
	mu          sync.RWMutex
)

// GetOnlineUsers retorna a lista de usuários online
func GetOnlineUsers() map[string]bool {
	mu.RLock()
	defer mu.RUnlock()
	// Retorna uma cópia para evitar modificações externas
	usersCopy := make(map[string]bool)
	for u, s := range onlineUsers {
		usersCopy[u] = s
	}
	return usersCopy
}

// SetUserOnline marca um usuário como online
func SetUserOnline(username string) {
	mu.Lock()
	defer mu.Unlock()
	onlineUsers[username] = true
	// Salva atividade no banco
	db.SaveUserActivity(username, "online")
}

// SetUserOffline marca um usuário como offline
func SetUserOffline(username string) {
	mu.Lock()
	defer mu.Unlock()
	delete(onlineUsers, username)
	// Salva atividade no banco
	db.SaveUserActivity(username, "offline")
}

// StatusHandler lida com as mensagens de status de usuário do MQTT
func StatusHandler(client mqtt.Client, msg mqtt.Message) {
	var status struct {
		Username string `json:"username"`
		Online   bool   `json:"online"`
	}
	if err := json.Unmarshal(msg.Payload(), &status); err != nil {
		log.Printf("Error unmarshaling status message: %v", err)
		return
	}

	if status.Online {
		SetUserOnline(status.Username)
		log.Printf("User %s is now online.", status.Username)
	} else {
		SetUserOffline(status.Username)
		log.Printf("User %s is now offline.", status.Username)
	}
}
