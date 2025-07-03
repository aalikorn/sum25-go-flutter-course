package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"lab03-backend/models"
	"lab03-backend/storage"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Handler holds the storage instance
type Handler struct {
	storage *storage.MemoryStorage
}

// NewHandler creates a new handler instance
func NewHandler(storage *storage.MemoryStorage) *Handler {
	return &Handler{storage: storage}
}

// SetupRoutes configures all API routes
func (h *Handler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Apply CORS middleware globally
	router.Use(corsMiddleware)

	apiRouter := router.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/messages", h.GetMessages).Methods(http.MethodGet)
	apiRouter.HandleFunc("/messages", h.CreateMessage).Methods(http.MethodPost)
	apiRouter.HandleFunc("/messages/{id}", h.UpdateMessage).Methods(http.MethodPut)
	apiRouter.HandleFunc("/messages/{id}", h.DeleteMessage).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/status/{code}", h.GetHTTPStatus).Methods(http.MethodGet)
	apiRouter.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)

	return router
}

// GetMessages handles GET /api/messages
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := h.storage.GetAllMessages()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get messages")
		return
	}
	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    messages,
	})
}

// CreateMessage handles POST /api/messages
func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req models.CreateMessageRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Content) == "" {
		h.writeError(w, http.StatusBadRequest, "Username and Content are required")
		return
	}

	msg, err := h.storage.CreateMessage(req.Username, req.Content)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to create message")
		return
	}

	h.writeJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    msg,
	})
}

// UpdateMessage handles PUT /api/messages/{id}
func (h *Handler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		h.writeError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	var req models.UpdateMessageRequest
	if err := h.parseJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		h.writeError(w, http.StatusBadRequest, "Content is required")
		return
	}

	msg, err := h.storage.UpdateMessage(id, req.Content)
	if err != nil {
		if errors.Is(err, storage.ErrMessageNotFound) {
			h.writeError(w, http.StatusNotFound, "Message not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Failed to update message")
		return
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    msg,
	})
}

// DeleteMessage handles DELETE /api/messages/{id}
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		h.writeError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	err = h.storage.DeleteMessage(id)
	if err != nil {
		if errors.Is(err, storage.ErrMessageNotFound) {
			h.writeError(w, http.StatusNotFound, "Message not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Failed to delete message")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetHTTPStatus handles GET /api/status/{code}
func (h *Handler) GetHTTPStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	codeStr := vars["code"]
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 100 || code > 599 {
		h.writeError(w, http.StatusBadRequest, "Invalid HTTP status code")
		return
	}

	resp := models.HTTPStatusResponse{
		StatusCode:  code,
		ImageURL:    fmt.Sprintf("https://http.cat/%d", code),
		Description: getHTTPStatusDescription(code),
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    resp,
	})
}

// HealthCheck handles GET /api/health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	count, err := h.storage.CountMessages()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get message count")
		return
	}

	resp := map[string]interface{}{
		"status":         "ok",
		"message":        "API is running",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"total_messages": count,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// Helper function to write JSON responses
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log encoding error
		fmt.Printf("JSON encoding error: %v\n", err)
	}
}

// Helper function to write error responses
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	resp := models.APIResponse{
		Success: false,
		Error:   message,
	}
	h.writeJSON(w, status, resp)
}

// Helper function to parse JSON request body
func (h *Handler) parseJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	return decoder.Decode(dst)
}

// Helper function to get HTTP status description
func getHTTPStatusDescription(code int) string {
	switch code {
	case 100:
		return "Continue"
	case 101:
		return "Switching Protocols"
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 202:
		return "Accepted"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return "Unknown Status"
	}
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
