package main

import (
	"log"
	"net/http"

	"svc.orchestrator/handlers"
	"svc.orchestrator/registry"
	"svc.orchestrator/storage"
)

func main() {
	session := storage.NewSession([]string{"127.0.0.1"})
	datastore := storage.NewDataStore(session)

	datastore.StartRollup()
	defer datastore.StopRollup()

	aggregator := registry.NewMetricsAggregator(datastore)
	aggregator.Start()
	defer aggregator.Stop()

	svcRegistry := registry.NewServiceRegistry(aggregator)
	apiManager := handlers.NewAPIManager(svcRegistry, datastore)

	apiManager.RegisterRoutes()
	svcRegistry.Start()

	err := http.ListenAndServe(":8500", nil)
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
