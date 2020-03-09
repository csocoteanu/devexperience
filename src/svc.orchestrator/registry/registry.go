package registry

import (
	"clients"
	"context"
	"errors"
	"log"
	"sync"

	"svc.orchestrator/types"
)

const maxHeartBeatRetries = 1

type serviceRegistry struct {
	healthCheckers        map[string][]*healthChecker
	healthCheckersLock    *sync.RWMutex
	healthCheckerExitChan chan types.RegistrantInfo
}

// NewServiceRegistry creates a new service registry instance
func NewServiceRegistry() *serviceRegistry {
	s := serviceRegistry{
		healthCheckers:        make(map[string][]*healthChecker),
		healthCheckerExitChan: make(chan types.RegistrantInfo),
		healthCheckersLock:    &sync.RWMutex{},
	}

	return &s
}

func (s *serviceRegistry) Register(ctx context.Context, req *clients.RegisterRequest) (*clients.RegisterResponse, error) {
	if len(req.ControlAddress) == 0 ||
		len(req.ServiceName) == 0 ||
		len(req.DataAddress) == 0 {
		return nil, errors.New("invalid fields")
	}

	rInfo := types.NewRegistrantInfo(req.ServiceName, req.ControlAddress, req.DataAddress)

	log.Printf("Received register request: %s", rInfo.String())

	err := s.load(rInfo)
	if err != nil {
		return nil, err
	}

	resp := clients.RegisterResponse{Code: clients.RegisterSuccess}
	return &resp, nil
}

func (s *serviceRegistry) Unregister(ctx context.Context, req *clients.RegisterRequest) (*clients.RegisterResponse, error) {
	log.Printf("Trying to unregister service=%s control=%s", req.ServiceName, req.ControlAddress)

	s.healthCheckersLock.RLock()
	defer s.healthCheckersLock.RUnlock()

	hCheckers, ok := s.healthCheckers[req.ServiceName]
	if !ok {
		log.Printf("Service with name=%s does not exist! Skipping...", req.ServiceName)
		return nil, errors.New("registrant missing")
	}

	for _, hChecker := range hCheckers {
		if hChecker.info.ControlAddress == req.ControlAddress {
			hChecker.stopHealthCheck()
		}
	}

	resp := clients.RegisterResponse{Code: clients.RegisterSuccess}
	return &resp, nil
}

func (s *serviceRegistry) GetServices() (map[string][]types.RegistrantInfo, error) {
	s.healthCheckersLock.RLock()
	defer s.healthCheckersLock.RUnlock()

	result := map[string][]types.RegistrantInfo{}

	for serviceName, hCheckers := range s.healthCheckers {
		rInfos := []types.RegistrantInfo{}
		for _, hChecker := range hCheckers {
			rInfos = append(rInfos, hChecker.info)
		}

		result[serviceName] = rInfos
	}

	return result, nil
}

func (s *serviceRegistry) Start() {
	go s.startRemoveHealthChecker()
}

func (s *serviceRegistry) Stop() {
	close(s.healthCheckerExitChan)
}

func (s *serviceRegistry) load(rInfos ...types.RegistrantInfo) error {
	s.healthCheckersLock.Lock()
	defer s.healthCheckersLock.Unlock()

	for _, rInfo := range rInfos {
		hCheckers, ok := s.healthCheckers[rInfo.ServiceName]
		if ok {
			for _, hChecker := range hCheckers {
				if hChecker.info.ControlAddress == rInfo.ControlAddress {
					log.Printf("Already registered service=%s address=%s", rInfo.ServiceName, rInfo.ControlAddress)
					return errors.New("registrant exists")
				}
			}
		}

		hChecker := newHealthChecker(rInfo, s.healthCheckerExitChan)
		s.healthCheckers[rInfo.ServiceName] = append(s.healthCheckers[rInfo.ServiceName], hChecker)

		log.Printf("Succesfully registered service=%s address=%s", rInfo.ServiceName, rInfo.ControlAddress)
	}

	return nil
}

func (s *serviceRegistry) startRemoveHealthChecker() {
	for {
		rInfo, ok := <-s.healthCheckerExitChan
		if !ok {
			log.Print("Exiting remove healthcheck listener")
		}

		s.removeHealthChecker(rInfo)
	}
}

func (s *serviceRegistry) removeHealthChecker(rInfo types.RegistrantInfo) {
	log.Printf("Removing healthchecker for %s", rInfo.String())

	s.healthCheckersLock.Lock()
	defer s.healthCheckersLock.Unlock()

	hCheckers, ok := s.healthCheckers[rInfo.ServiceName]
	if !ok {
		log.Printf("Skipping missing service name=%s", rInfo.ServiceName)
		return
	}

	remaining := []*healthChecker{}
	for _, hChecker := range hCheckers {
		if hChecker.info.ControlAddress == rInfo.ControlAddress {
			log.Printf("Succesfully unregistered service=%s control=%s", rInfo.ServiceName, rInfo.ControlAddress)
		} else {
			remaining = append(remaining, hChecker)
		}
	}

	if len(remaining) == 0 {
		delete(s.healthCheckers, rInfo.ServiceName)
	} else {
		s.healthCheckers[rInfo.ServiceName] = remaining
	}
}
