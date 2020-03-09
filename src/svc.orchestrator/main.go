package main

import (
	"log"
	"net/http"

	"svc.orchestrator/handlers"

	"svc.orchestrator/registry"
)

func main() {
	svcRegistry := registry.NewServiceRegistry()
	apiManager := handlers.NewAPIManager(svcRegistry)

	apiManager.RegisterRoutes()
	svcRegistry.Start()

	err := http.ListenAndServe(":8500", nil)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
