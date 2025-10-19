package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"event_calendar/pkg/telegram"
)

type WebhookUpdate struct {
	UpdateID      int                    `json:"update_id"`
	CallbackQuery *telegram.CallbackQuery `json:"callback_query,omitempty"`
	Message       map[string]interface{} `json:"message,omitempty"`
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	telegramService := telegram.NewService(botToken)

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var update WebhookUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			log.Printf("Error decoding webhook: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Handle callback queries (button presses)
		if update.CallbackQuery != nil {
			log.Printf("Received callback query: %s", update.CallbackQuery.Data)
			
			if err := telegramService.HandleCallbackQuery(*update.CallbackQuery); err != nil {
				log.Printf("Error handling callback query: %v", err)
			}
		}

		// Handle regular messages
		if update.Message != nil {
			log.Printf("Received message: %v", update.Message)
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Webhook server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
