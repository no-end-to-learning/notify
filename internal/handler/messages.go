package handler

import (
	"encoding/json"
	"net/http"

	"notify/internal/queue"
	"notify/internal/service"
)

type SendMessageRequest struct {
	Channel string                `json:"channel"`
	Target  string                `json:"target"`
	Params  service.MessageParams `json:"params"`
}

type SendRawMessageRequest struct {
	Channel string         `json:"channel"`
	Target  string         `json:"target"`
	Message map[string]any `json:"message"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func SendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if req.Channel == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel and target are required")
		return
	}

	channel, err := service.ValidateChannel(req.Channel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	svc, err := service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	message := svc.BuildMessage(req.Params)
	queue.GetManager().Enqueue(channel, req.Target, message)

	writeJSON(w, http.StatusOK, &service.SendResult{
		Success: true,
	})
}

func SendRawMessage(w http.ResponseWriter, r *http.Request) {
	var req SendRawMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if req.Channel == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel and target are required")
		return
	}

	channel, err := service.ValidateChannel(req.Channel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	_, err = service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	queue.GetManager().Enqueue(channel, req.Target, req.Message)

	writeJSON(w, http.StatusOK, &service.SendResult{
		Success: true,
	})
}

func ListChats(w http.ResponseWriter, r *http.Request) {
	channelStr := r.URL.Query().Get("channel")
	if channelStr == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel is required")
		return
	}

	if channelStr != "lark" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "only lark channel supports listing chats")
		return
	}

	channel, _ := service.ValidateChannel(channelStr)
	svc, err := service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	lister, ok := svc.(service.ChatLister)
	if !ok {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel does not support listing chats")
		return
	}

	chats, err := lister.ListChats()
	if err != nil {
		writeError(w, http.StatusBadGateway, "SERVICE_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, chats)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   code,
		Message: message,
	})
}
