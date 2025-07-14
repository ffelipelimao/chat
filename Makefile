# Makefile for Chat-PoC Project

.PHONY: all build run-backend run-tester clean help

# Default target
all: help

# Build the main application
build:
	@echo "Building backend application..."
	@go build -o chat-app main.go
	@echo "Build complete: ./chat-app"

# Run the backend server
run-backend:
	@echo "Starting backend server..."
	@go run main.go

# Run the MQTT tester script with parameters
# Usage: make run-tester MSG="your message" [USER="user name"] [TOPIC="your/topic"]
MSG ?= "default test message"
USER ?= "script-tester"
TOPIC ?= "chat/messages"
run-tester:
	@echo "Running MQTT tester..."
	@go run tools/mqtt_tester.go -message="$(MSG)" -user="$(USER)" -topic="$(TOPIC)"

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -f chat-app
	@echo "Cleanup complete."

# Help target to display usage
help:
	@echo "Available commands:"
	@echo "  make build          - Build the backend application"
	@echo "  make run-backend    - Run the backend server"
	@echo "  make run-tester     - Run the MQTT tester with default values"
	@echo "  make run-tester MSG=\\"your message\\" [USER=\\"user name\\"] - Run the MQTT tester with custom message and user"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make help           - Show this help message" 