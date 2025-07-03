package main

import (
	"log"
	"net/http"
	"time"

	"lab03-backend/api"
	"lab03-backend/storage"
)

func main() {
	// Создание хранилища и хендлера
	store := storage.NewMemoryStorage()
	handler := api.NewHandler(store)
	router := handler.SetupRoutes()

	// Настройки HTTP-сервера
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("[✅] Server starting on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[🔥] Server failed: %v", err)
	}
}
