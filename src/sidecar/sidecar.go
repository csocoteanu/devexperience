package sidecar

import (
	"clients"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/pkg/errors"
)

type Proxy struct {
	controlAddress      string
	orchestratorAddress string
	serviceName         string
	dataAddress         string
	lastUpdatedTime     time.Time
	lastUpdatedLock     *sync.Mutex
	client              clients.OrchestratorClient
}

// NewProxy creates a new sidecar instance
func NewProxy(
	orchestratorAddress string,
	controlAddress string,
	serviceName, serviceLocalAddress string) *Proxy {

	client := clients.NewOrchestratorClient(orchestratorAddress)

	s := Proxy{
		serviceName:         serviceName,
		dataAddress:         serviceLocalAddress,
		controlAddress:      controlAddress,
		orchestratorAddress: orchestratorAddress,
		lastUpdatedLock:     &sync.Mutex{},
		client:              client,
	}

	log.Printf("Creating sidecar: %s", s.String())

	return &s
}

func (s *Proxy) Start() {
	go s.connectToOrchestrator()

	go func() {
		if err := s.listenForHeartBeats(); err != nil {
			log.Fatalf("Failed starting sidecar on %s err=%s", s.controlAddress, err.Error())
		}
	}()
}

func (s *Proxy) String() string {
	return fmt.Sprintf("[%s] data=%s control=%s", s.serviceName, s.dataAddress, s.controlAddress)
}

func (s *Proxy) register() error {
	log.Printf("Registering to service=%s control address=%s", s.orchestratorAddress, s.controlAddress)

	var err error
	var resp *clients.RegisterResponse
	var expRetrier = retrier.New(retrier.ExponentialBackoff(4, 500*time.Millisecond), nil)

	if err = expRetrier.Run(func() error {
		req := clients.RegisterRequest{
			ControlAddress: fmt.Sprintf("http://%s", s.controlAddress),
			ServiceName:    s.serviceName,
			DataAddress:    s.dataAddress,
		}

		resp, err = s.client.RegisterSidecar(context.Background(), &req)
		if err != nil {
			log.Printf("Error registering to %s! err=%s", s.orchestratorAddress, err.Error())
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if resp.Code != clients.RegisterSuccess {
		log.Printf("Error received when registering to %s! err=%s", s.orchestratorAddress, resp.ErrMessage)
		return errors.New(resp.ErrMessage)
	}

	s.setUpdatedTime()

	return nil
}

func (s *Proxy) listenForHeartBeats() error {
	log.Printf("Starting sidecar on address=%s", s.controlAddress)

	http.HandleFunc(clients.ProxyHealthURL, s.handleHeartbeat)
	return http.ListenAndServe(s.controlAddress, nil)
}

func (s *Proxy) connectToOrchestrator() {
	defer func() { log.Printf("Succesfully registered service=%s", s.serviceName) }()

	if err := s.register(); err != nil {
		log.Printf("Failed registering! err=%s", err.Error())
	}

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		register := false

		s.lastUpdatedLock.Lock()
		duration := time.Now().Sub(s.lastUpdatedTime)
		if duration.Minutes() > 1 {
			register = true
		}
		s.lastUpdatedLock.Unlock()

		if register {
			if err := s.register(); err != nil {
				log.Printf("Failed registering! err=%s", err.Error())
			}
		}
	}
}

func (s *Proxy) handleHeartbeat(w http.ResponseWriter, req *http.Request) {
	log.Printf("Received heartbeat: %s", s.String())

	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.setUpdatedTime()

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	hostname, _ := os.Hostname()

	resp := clients.HeartbeatResponse{
		Stats: []clients.Stats{
			{
				TS:            time.Now().UTC(),
				ServiceID:     s.serviceName + hostname,
				CPU:           20,
				Mem:           float64(memStats.Sys),
				Threads:       float64(runtime.NumGoroutine()),
				NumGoroutines: float64(runtime.NumGoroutine()),
			},
		},
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Unable to marshall heartbeat response! error=%+v in %s", err, s.String())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(respBytes)
	if err != nil {
		log.Printf("Unable to send response! error=%+v in %s", err, s.String())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Proxy) setUpdatedTime() {
	s.lastUpdatedLock.Lock()
	s.lastUpdatedTime = time.Now()
	s.lastUpdatedLock.Unlock()
}
