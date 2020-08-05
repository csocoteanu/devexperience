package registry

import (
	"clients"
	"fmt"
	"log"
	"time"

	"svc.orchestrator/storage"
)

type MetricsAggregator struct {
	metrics map[string]*clients.Aggregation
	c       chan *clients.DataPoint
	done    chan bool
	ticker  *time.Ticker
	store   *storage.DataStore
}

func NewMetricsAggregator(store *storage.DataStore) *MetricsAggregator {
	return &MetricsAggregator{
		store:   store,
		metrics: make(map[string]*clients.Aggregation),
		c:       make(chan *clients.DataPoint, 10000),
		done:    make(chan bool),
	}
}

func (a *MetricsAggregator) AddDataPoint(dp *clients.DataPoint) {
	a.c <- dp
}

func (a *MetricsAggregator) run() {
	t := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-a.done:
			return
		case <-t.C:
			log.Println("Storing metric aggregation")
			err := a.store.InsertAggregations(a.metrics)
			if err != nil {
				log.Println(err)
			}
			a.metrics = make(map[string]*clients.Aggregation)
		case dp := <-a.c:
			if metric, ok := a.metrics[getAggregationKey(dp.ServiceID, dp.MetricID)]; !ok {
				a.metrics[getAggregationKey(dp.ServiceID, dp.MetricID)] = &clients.Aggregation{
					MetricID:  dp.MetricID,
					ServiceID: dp.ServiceID,
					TS:        dp.TS,
					Max:       dp.Value,
					Min:       dp.Value,
					Average:   dp.Value,
					NumValues: 1,
				}
			} else {
				metric.Average = (dp.Value + float64(metric.NumValues)*metric.Average) / float64(metric.NumValues+1)
				metric.NumValues += 1
				if metric.Min > dp.Value {
					metric.Min = dp.Value
				}
				if metric.Max < dp.Value {
					metric.Max = dp.Value
				}
			}
		}
	}
}

func (a *MetricsAggregator) Start() {
	a.done = make(chan bool)
	a.ticker = time.NewTicker(1 * time.Minute)
	go a.run()
}

func (a *MetricsAggregator) Stop() {
	a.ticker.Stop()
	a.done <- true
}

func getAggregationKey(hostname, metricID string) string {
	return fmt.Sprintf("%s:%s", hostname, metricID)
}
