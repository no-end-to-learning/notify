package handler

import (
	"encoding/json"
	"io"
	"log/slog"
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to read request body")
		return
	}
	slog.Info("Send message request received body=" + string(body))

	var req SendMessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	channel, svc, ok := resolveService(w, req.Channel, req.Target)
	if !ok {
		return
	}

	message := svc.BuildMessage(req.Params)
	queue.GetManager().Enqueue(channel, req.Target, message)

	writeJSON(w, http.StatusOK, &service.SendResult{Success: true})
}

func SendRawMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to read request body")
		return
	}
	slog.Info("Send raw message request received body=" + string(body))

	var req SendRawMessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	channel, _, ok := resolveService(w, req.Channel, req.Target)
	if !ok {
		return
	}

	queue.GetManager().Enqueue(channel, req.Target, req.Message)

	writeJSON(w, http.StatusOK, &service.SendResult{Success: true})
}

func ListChats(w http.ResponseWriter, r *http.Request) {
	channelStr := r.URL.Query().Get("channel")
	if channelStr == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel is required")
		return
	}

	channel, err := service.ValidateChannel(channelStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

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

// resolveService validates channel/target and returns the corresponding service.
// Writes an error response and returns false if validation fails.
func resolveService(w http.ResponseWriter, channelStr, target string) (service.Channel, service.NotifyService, bool) {
	if channelStr == "" || target == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel and target are required")
		return "", nil, false
	}

	channel, err := service.ValidateChannel(channelStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return "", nil, false
	}

	svc, err := service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return "", nil, false
	}

	return channel, svc, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		http.Error(w, `{"error":"INTERNAL_ERROR","message":"response serialization failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Error: code, Message: message})
}
