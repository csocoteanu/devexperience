package handlers

import (
	"clients"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"svc.orchestrator/types"
)

type APIManager struct {
	registry types.ServiceRegistry
}

func NewAPIManager(registry types.ServiceRegistry) *APIManager {
	m := APIManager{
		registry: registry,
	}

	return &m
}

func (m *APIManager) RegisterRoutes() {
	http.HandleFunc(clients.RegisterURL, m.handleRegister)
}

func (m *APIManager) handleRegister(w http.ResponseWriter, req *http.Request) {
	log.Printf("Handling register!")

	if req.Method != http.MethodPost {
		log.Printf("Got unsupported method=%s", req.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registerReq := clients.RegisterRequest{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&registerReq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	registerResp, err := m.registry.Register(context.Background(), &registerReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respBytes, err := json.Marshal(registerResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(respBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
