package syncapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"inventory-desktop/internal/backend/store"
)

func NewSyncHTTPHandler(s *store.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		health, err := s.Health()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, health)
	})

	mux.HandleFunc("/sync/push", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req store.SyncPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
			return
		}
		req.DeviceID = strings.TrimSpace(req.DeviceID)
		if req.DeviceID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "deviceId is required"})
			return
		}

		resp, err := s.ApplyIncomingMutations(req.DeviceID, req.Mutations)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/sync/pull", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		since := strings.TrimSpace(r.URL.Query().Get("since"))
		limit := int64(200)
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			v, err := strconv.ParseInt(rawLimit, 10, 64)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be integer"})
				return
			}
			limit = v
		}

		mutations, err := s.PullMutationsForDevice(deviceID, since, limit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, store.SyncPullResponse{Mutations: mutations})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
