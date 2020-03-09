package types

import (
	"clients"
	"context"
	"fmt"
)

// RegistrantInfo ...
type RegistrantInfo struct {
	ControlAddress string
	DataAddress    string
	ServiceName    string
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
