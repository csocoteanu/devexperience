package clients

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type orchestratorClient struct {
	client  *http.Client
	address string
}

// NewOrchestratorClient creates a new orchestrator client
func NewOrchestratorClient(address string) OrchestratorClient {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			DisableCompression:    false,
			DisableKeepAlives:     false,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       5 * time.Minute,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
		},
		Timeout: 5 * time.Minute,
	}

	c := orchestratorClient{
		client:  client,
		address: address,
	}

	return &c
}

func (c *orchestratorClient) RegisterSidecar(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	url := fmt.Sprintf("%s%s", c.address, RegisterURL)
	httpReq, err := toHTTPRequest(ctx, http.MethodPost, url, *req)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	resp := RegisterResponse{}

	switch httpResp.StatusCode {
	case http.StatusConflict:
		resp.Code = RegisterFailed
		resp.ErrMessage = "Registrant already exists!"
	case http.StatusOK:
		resp.Code = RegisterSuccess
	default:
		resp.Code = RegisterFailed
		resp.ErrMessage = fmt.Sprintf("Unknown error! HTTP CODE=%d", httpResp.StatusCode)
	}

	return &resp, nil
}
