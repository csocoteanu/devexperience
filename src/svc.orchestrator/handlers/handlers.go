package handlers

import (
	"clients"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"svc.orchestrator/storage"
	"svc.orchestrator/types"
	"time"
)

type APIManager struct {
	registry  types.ServiceRegistry
	dataStore *storage.DataStore
}

func NewAPIManager(registry types.ServiceRegistry, dataStore *storage.DataStore) *APIManager {
	m := APIManager{
		registry:  registry,
		dataStore: dataStore,
	}

	return &m
}

func (m *APIManager) RegisterRoutes() {
	http.HandleFunc(clients.RegisterURL, m.handleRegister)
	http.HandleFunc(clients.ServicesURL, m.handleGetServices)
	http.HandleFunc(clients.StatsURL, m.handleGetStats)
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

func (m *APIManager) handleGetServices(w http.ResponseWriter, req *http.Request) {
	log.Printf("Handling get services!")

	if req.Method != http.MethodGet {
		log.Printf("Got unsupported method=%s", req.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	services, err := m.registry.GetServices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	servicesResp := types.ServiceInfos{}

	for serviceName, sidecars := range services {
		si := types.ServiceInfo{
			ServiceName: serviceName,
			Registrants: sidecars,
		}
		servicesResp.Services = append(servicesResp.Services, si)
	}

	respBytes, err := json.Marshal(servicesResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(respBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *APIManager) handleGetStats(w http.ResponseWriter, req *http.Request) {
	log.Printf("Handling get stats!")

	if req.Method != http.MethodGet {
		log.Printf("Got unsupported method=%s", req.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricID, ok := req.URL.Query()["metricID"]
	if !ok || len(metricID[0]) < 1 {
		http.Error(w, "metricID is missing", http.StatusBadRequest)
		return
	}

	startTS, ok := req.URL.Query()["startTS"]
	if !ok || len(startTS[0]) < 1 {
		http.Error(w, "startTS is missing", http.StatusBadRequest)
		return
	}

	fmt.Println(startTS[0])

	sts, err := strconv.ParseInt(startTS[0], 10, 64)
	if err != nil {
		log.Println(err)
		http.Error(w, "startTS is not a valid unix timestamp", http.StatusBadRequest)
		return
	}

	ets := time.Now().UTC()

	endTS, ok := req.URL.Query()["endTS"]
	if ok && len(endTS[0]) > 0 {
		unixEndTS, err := strconv.ParseInt(endTS[0], 64, 10)
		if err != nil {
			http.Error(w, "endTS is not a valid unix timestamp", http.StatusBadRequest)
			return
		}
		ets = time.Unix(unixEndTS, 0)
	}

	stats, err := m.dataStore.GetStats(metricID[0], time.Unix(sts, 0), ets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respBytes, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(respBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
