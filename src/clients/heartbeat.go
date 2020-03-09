package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type heartbeatClient struct {
	client  *http.Client
	address string
}

// NewHeartbeatClient creates a new heartbeat client
func NewHeartbeatClient(address string) HeartbeatClient {
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

	c := heartbeatClient{
		client:  client,
		address: address,
	}

	return &c
}

// Heartbeat ...
func (c *heartbeatClient) Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	url := fmt.Sprintf("%s%s", c.address, ProxyHealthURL)
	httpReq, err := toHTTPRequest(ctx, http.MethodGet, url, *req)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error encountered! status=%d", httpResp.StatusCode)
	}

	result := HeartbeatResponse{}
	d := json.NewDecoder(httpResp.Body)
	d.UseNumber()
	if err := d.Decode(result); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal heartbeat response")
	}

	return &result, nil
}
