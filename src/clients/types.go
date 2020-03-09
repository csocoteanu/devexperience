package clients

import "context"

const (
	RegisterSuccess = iota
	RegisterFailed
)

const (
	ProxyHealthURL = "/health"
	RegisterURL    = "/register"
)

// HeartbeatRequest TODO: add filter params
type HeartbeatRequest struct{}

// HeartbeatResponse is the json returned by the sidecars to the service orchestrator
type HeartbeatResponse struct {
	Stats []Stats `json:"stats"`
}

// Stats is the json containing relevant info about the host
type Stats struct {
	CPU     int `json:"cpu"`
	Mem     int `json:"mem"`
	Threads int `json:"threads"`
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
