package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

func toHTTPRequest(
	ctx context.Context,
	method string,
	endpoint string,
	body interface{}) (*http.Request, error) {

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	return req, nil
}
