package clients

import (
	"context"
	"time"
)

const (
	RegisterSuccess = iota
	RegisterFailed
)

const (
	ProxyHealthURL = "/health"
	RegisterURL    = "/register"
	ServicesURL    = "/services"
	StatsURL       = "/stats"
)

// HeartbeatRequest TODO: add filter params
type HeartbeatRequest struct{}

// HeartbeatResponse is the json returned by the sidecars to the service orchestrator
type HeartbeatResponse struct {
	Stats []Stats `json:"stats"`
}

type Aggregation struct {
	MetricID  string    `json:"metric_id"`
	TS        time.Time `json:"time"`
	ServiceID string    `json:"service_id"`
	Max       float64   `json:"max"`
	Min       float64   `json:"min"`
	Average   float64   `json:"avg"`
	NumValues int       `json:"val"`
}

type DataPoint struct {
	MetricID  string
	TS        time.Time
	ServiceID string
	Value     float64
}

// Stats is the json containing relevant info about the host
type Stats struct {
	TS            time.Time `json:"time"`
	ServiceID     string    `json:"service_id"`
	CPU           float64   `json:"cpu"`
	Mem           float64   `json:"mem"`
	Threads       float64   `json:"threads"`
	NumGoroutines float64   `json:"num_goroutines"`
}

// RegisterRequest is the message sent by the host to the orchestrator
type RegisterRequest struct {
	ControlAddress string `json:"control_address"`
	DataAddress    string `json:"data_address"`
	ServiceName    string `json:"service_name"`
}

// RegisterResponse is the message sent by the orchestrator to the host
type RegisterResponse struct {
	Code       int    `json:"code"`
	ErrMessage string `json:"err_message"`
}

// OrchestratorClient interface for interacting with the orchestrator service
type OrchestratorClient interface {
	RegisterSidecar(context.Context, *RegisterRequest) (*RegisterResponse, error)
}

type HeartbeatClient interface {
	Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
}
