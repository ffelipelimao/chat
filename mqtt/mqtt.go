package mqtt

import (
	"encoding/json"
	"log"
	"time"

	"chat-poc/entity"
	userstatus "chat-poc/user_status"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttClient mqtt.Client

const (
	mqttBroker  = "tcp://localhost:1883"
	chatTopic   = "chat/messages"
	statusTopic = "chat/status"
)

func InitMQTTClient(messageHandler mqtt.MessageHandler) {
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker).SetClientID("go-chat-server")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	opts.SetDefaultPublishHandler(messageHandler)

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT broker!")
		// Subscreve ao tópico de mensagens
		token := c.Subscribe(chatTopic, 1, messageHandler)
		token.Wait()
		if token.Error() != nil {
			log.Printf("Error subscribing to chat topic: %v\n", token.Error())
		} else {
			log.Printf("Subscribed to topic: %s\n", chatTopic)
		}

		token = c.Subscribe(statusTopic, 1, userstatus.StatusHandler)
		token.Wait()
		if token.Error() != nil {
			log.Printf("Error subscribing to status topic: %v\n", token.Error())
		} else {
			log.Printf("Subscribed to topic: %s\n", statusTopic)
		}
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT Connection Lost: %v", err)
	}

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error connecting to MQTT broker: %v", token.Error())
	}
}

func PublishMessage(username, content string) error {
	msg := entity.Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.Printf("Publishing message: %s", string(payload))

	token := mqttClient.Publish(chatTopic, 1, false, payload)
	token.Wait()
	return token.Error()
}

func PublishUserStatus(username string, online bool) {
	status := map[string]interface{}{
		"username": username,
		"online":   online,
	}
	payload, err := json.Marshal(status)
	if err != nil {
		log.Printf("Error marshaling user status: %v", err)
		return
	}
	token := mqttClient.Publish(statusTopic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error publishing user status: %v", token.Error())
	}
}
