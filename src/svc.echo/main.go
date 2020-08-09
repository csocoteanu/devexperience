package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sidecar"
)

var controlPort, appLocalPort *int

type EchoRequest struct {
	Message string `json:"message"`
}

func parseArgs() {
	controlPort = flag.Int("control-port", 8060, "Control port")
	appLocalPort = flag.Int("app-port", 10010, "Application port")

	flag.Parse()
}

func main() {
	parseArgs()

	log.Printf("Starting ECHO server on data port=%d control port=%d", *appLocalPort, *controlPort)

	// start control plane
	proxy := sidecar.NewProxy(
		"http://localhost:8500",
		fmt.Sprintf("localhost:%d", *controlPort),
		"echo",
		fmt.Sprintf("http://localhost:%d", *appLocalPort))
	proxy.Start()

	http.HandleFunc("/echo", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Method not allowed"))
			return
		}

		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Failed to read body: %+v", err)))
			return
		}
		defer req.Body.Close()

		echoRequest := &EchoRequest{}
		err = json.Unmarshal(data, echoRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Failed to parse request: %+v", err)))
			return
		}

		log.Printf("Received message=%s", echoRequest.Message)
		data, err = json.Marshal(echoRequest)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Failed to marshal response: %+v", err)))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)

	})

	// start data plane
	err := http.ListenAndServe(fmt.Sprintf(":%d", *appLocalPort), nil)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
