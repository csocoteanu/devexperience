package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sidecar"
)

var controlPort, appLocalPort *int

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

	// start data plane
	err := http.ListenAndServe(fmt.Sprintf(":%d", *appLocalPort), nil)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
