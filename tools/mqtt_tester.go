package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ChatMessage define a estrutura do payload da mensagem
type ChatMessage struct {
	Username  string    `json:"Username"`
	Content   string    `json:"Content"`
	Timestamp time.Time `json:"Timestamp"`
}

func main() {
	// Definição dos flags de linha de comando
	message := flag.String("message", "", "A mensagem a ser enviada.")
	user := flag.String("user", "script-tester", "O nome de usuário para enviar a mensagem.")
	topic := flag.String("topic", "chat/messages", "O tópico para o qual enviar a mensagem.")
	broker := flag.String("broker", "tcp://localhost:1883", "O endereço do broker MQTT.")
	flag.Parse()

	// Validação dos argumentos
	if *message == "" {
		fmt.Println("Erro: O argumento -message é obrigatório.")
		os.Exit(1)
	}

	// Configuração do cliente MQTT
	opts := mqtt.NewClientOptions().AddBroker(*broker)
	opts.SetClientID("go-mqtt-publisher")
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Erro ao conectar ao broker: %v\n", token.Error())
		os.Exit(1)
	}
	fmt.Println("Conectado ao broker MQTT.")

	// Cria o payload da mensagem
	chatMsg := ChatMessage{
		Username:  *user,
		Content:   *message,
		Timestamp: time.Now(),
	}
	payload, err := json.Marshal(chatMsg)
	if err != nil {
		fmt.Printf("Erro ao criar o payload JSON: %v\n", err)
		os.Exit(1)
	}

	// Publica a mensagem
	token := client.Publish(*topic, 0, false, payload)
	token.Wait()
	if token.Error() != nil {
		fmt.Printf("Erro ao publicar mensagem: %v\n", token.Error())
	} else {
		fmt.Printf("Mensagem de %s publicada no tópico '%s': %s\n", *user, *topic, string(payload))
	}

	// Desconecta do broker
	client.Disconnect(250)
	fmt.Println("Desconectado do broker.")
}
