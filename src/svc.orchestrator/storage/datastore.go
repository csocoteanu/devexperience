package storage

import (
	"clients"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/pkg/errors"
)

// https://www.datastax.com/blog/2012/05/metric-collection-and-storage-cassandra

const (
	insertDataPointStmt = "INSERT INTO metrics (metric_id, ts, service_id, min, max, avg) VALUES (?, ?, ?, ?, ?, ?)"
	selectDataPointStmt = "SELECT ts, service_id, min, max, avg FROM metrics WHERE metric_id = ? AND ts > ?"
	insertRollup120Stmt = "INSERT INTO rollups120 (metric_id, ts, service_id, min, max, avg) VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS"
	insertRollup300Stmt = "INSERT INTO rollups300 (metric_id, ts, service_id, min, max, avg) VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS"
	selectRollup300Stmt = "SELECT metric_id, ts, service_id, min, max, avg FROM rollups300 WHERE metric_id = ? AND ts >= ?"
)

const (
	MetricCPU          = "cpu"
	MetricMemory       = "mem"
	MetricThreads      = "threads"
	MetricNumGoroutine = "num_goroutines"
)

var resourceMetrics = []string{MetricCPU, MetricMemory, MetricThreads, MetricNumGoroutine}

type DataStore struct {
	session *gocql.Session
	done    chan bool
}

func NewSession(seeds []string) *gocql.Session {
	cluster := gocql.NewCluster(seeds...)
	cluster.Keyspace = "stats"
	cluster.NumConns = 10
	cluster.MaxPreparedStmts = 1000
	cluster.MaxRoutingKeyInfo = 1000
	cluster.ConnectTimeout = 5 * time.Second
	cluster.ReconnectInterval = 5 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}

	return session
}

func NewDataStore(session *gocql.Session) *DataStore {
	return &DataStore{
		session: session,
		done:    make(chan bool),
	}
}

func (d *DataStore) StartRollup() {
	go d.startRollup()
}

func (d *DataStore) StopRollup() {
	d.done <- true
}

func (d *DataStore) InsertAggregations(aggs map[string]*clients.Aggregation) error {
	if len(aggs) == 0 {
		return nil
	}
	batch := gocql.NewBatch(gocql.LoggedBatch)
	for _, agg := range aggs {
		batch.Query(insertDataPointStmt, agg.MetricID, agg.TS, agg.ServiceID, agg.Min, agg.Max, agg.Average)
	}

	err := d.session.ExecuteBatch(batch)
	if err != nil {
		return errors.Wrapf(err, "Failed to store aggregations")
	}

	return nil
}

func (d *DataStore) InsertDataPoint(agg *clients.Aggregation) error {
	// insert a data point
	return d.session.Query(insertDataPointStmt, agg.MetricID, agg.TS, agg.Min, agg.Max, agg.Average).Exec()
}

func (d *DataStore) selectRawDataQuery(metricID string) *gocql.Iter {
	return d.session.Query(selectDataPointStmt, metricID, time.Now().Add(-10*time.Minute)).Iter()
}

func (d *DataStore) insertRollup300(agg *clients.Aggregation) error {
	return d.session.Query(insertRollup300Stmt, agg.MetricID, agg.TS, agg.ServiceID, agg.Min, agg.Max, agg.Average).Exec()
}

func (d *DataStore) insertRollup120(agg *clients.Aggregation) error {
	return d.session.Query(insertRollup120Stmt, agg.MetricID, agg.TS, agg.ServiceID, agg.Min, agg.Max, agg.Average).Exec()
}

func (d *DataStore) startRollup() error {
	rollup120Timer := time.NewTicker(2 * time.Minute)
	rollup300Timer := time.NewTicker(5 * time.Minute)

	for {
		select {
		case <-d.done:
			rollup300Timer.Stop()
			return nil
		case <-rollup300Timer.C: // aggregation
			log.Println("Running rollup300")
			wg := sync.WaitGroup{}
			for _, metricID := range resourceMetrics {
				wg.Add(1)
				func(metricID string) {
					defer wg.Done()
					err := d.runRollup(metricID, 300, d.selectRawDataQuery, d.insertRollup300)
					if err != nil {
						log.Println(err)
					}
				}(metricID)
				// TODO add rollups7200
				// TODO add rollups86400
				// ...
			}
			wg.Wait()
		case <-rollup120Timer.C:
			log.Println("Running rollup300")
			wg := sync.WaitGroup{}
			for _, metricID := range resourceMetrics {
				wg.Add(1)
				go func(metricID string) {
					defer wg.Done()
					err := d.runRollup(metricID, 120, d.selectRawDataQuery, d.insertRollup120)
					if err != nil {
						log.Println(err)
					}
				}(metricID)
			}
			wg.Wait()
		}
	}
}

func (d *DataStore) runRollup(metricID string, aggInterval int64, queryPreviousRollup func(metricID string) *gocql.Iter, storeAggregation func(agg *clients.Aggregation) error) error {
	metrics := make(map[string]*clients.Aggregation)
	latestInterval := make(map[string]int64)
	for {
		var ts time.Time
		var serviceID string
		var min, max, avg float64
		query := queryPreviousRollup(metricID)
		for {
			exists := query.Scan(&ts, &serviceID, &min, &max, &avg)
			if !exists {
				return nil
			}

			aggregationKey := getAggregationKey(serviceID, metricID)

			interval := ts.UTC().Unix() / aggInterval
			if li, ok := latestInterval[aggregationKey]; ok && li < interval {
				if agg, ok := metrics[aggregationKey]; ok {
					err := storeAggregation(agg)
					if err != nil {
						return err
					}
				}

				delete(metrics, aggregationKey)
				delete(latestInterval, aggregationKey)
			}

			// agg each value found in db
			if agg, ok := metrics[aggregationKey]; !ok {
				latestInterval[aggregationKey] = interval
				metrics[aggregationKey] = &clients.Aggregation{
					MetricID:  metricID,
					ServiceID: serviceID,
					TS:        time.Unix(interval*aggInterval, 0),
					Min:       min,
					Max:       max,
					Average:   avg,
					NumValues: 1,
				}
			} else {
				agg.Average = (avg + float64(agg.NumValues)*agg.Average) / float64(agg.NumValues+1)
				agg.NumValues += 1
				if agg.Min > min {
					agg.Min = min
				}
				if agg.Max < max {
					agg.Max = max
				}
			}
		}
	}
}

func (d *DataStore) GetResourceStats(startTS, endTS time.Time) ([]clients.Aggregation, error) {
	aggregations := make([]clients.Aggregation, 0, 5)
	for _, metricID := range resourceMetrics {
		agg, err := d.GetStats(metricID, startTS, endTS)
		if err != nil {
			return aggregations, err
		}
		aggregations = append(aggregations, agg...)
	}
	return aggregations, nil
}

func (d *DataStore) GetStats(metricID string, startTS, endTs time.Time) ([]clients.Aggregation, error) {
	aggregations := make([]clients.Aggregation, 0, 10)
	iter := d.session.Query(selectRollup300Stmt, metricID, startTS).Iter()
	for {
		agg := clients.Aggregation{}
		exists := iter.Scan(&agg.MetricID, &agg.TS, &agg.ServiceID, &agg.Min, &agg.Max, &agg.Average)
		if !exists {
			break
		}
		aggregations = append(aggregations, agg)
	}
	return aggregations, nil
}

func getAggregationKey(serviceID, metricID string) string {
	return fmt.Sprintf("%s:%s", serviceID, metricID)
}
