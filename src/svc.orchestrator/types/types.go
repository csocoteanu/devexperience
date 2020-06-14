package types

import (
	"clients"
	"context"
	"fmt"
)

// ServiceInfos ...
type ServiceInfos struct {
	Services []ServiceInfo `json:"services"`
}

// ServiceInfo ...
type ServiceInfo struct {
	ServiceName string           `json:"service_name"`
	Registrants []RegistrantInfo `json:"registrants"`
}

// RegistrantInfo ...
type RegistrantInfo struct {
	ControlAddress string `json:"control_address"`
	DataAddress    string `json:"data_address"`
	ServiceName    string `json:"service_name"`
}

// NewRegistrantInfo creates a new registrant info instance
func NewRegistrantInfo(serviceName, controlAddress, dataAddress string) RegistrantInfo {
	ri := RegistrantInfo{
		ControlAddress: controlAddress,
		ServiceName:    serviceName,
		DataAddress:    dataAddress,
	}

	return ri
}

func (ri *RegistrantInfo) String() string {
	return fmt.Sprintf("[%s] control=%s data=%s",
		ri.ServiceName,
		ri.ControlAddress,
		ri.DataAddress)
}

type ServiceRegistry interface {
	Register(ctx context.Context, req *clients.RegisterRequest) (*clients.RegisterResponse, error)
	Unregister(ctx context.Context, req *clients.RegisterRequest) (*clients.RegisterResponse, error)
	GetServices() (map[string][]RegistrantInfo, error)
	Start()
	Stop()
}
