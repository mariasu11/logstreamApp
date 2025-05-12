package metrics

import (
        "sync"

        "github.com/prometheus/client_golang/prometheus"
        "github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics for the application
type Metrics struct {
        // Log Processing Metrics
        LogBatchesReceived prometheus.Counter
        LogEntriesReceived prometheus.Counter
        LogEntriesProcessed prometheus.Counter
        LogEntriesFiltered prometheus.Counter
        LogEntriesErrored prometheus.Counter
        LogProcessingTime prometheus.Histogram

        // Worker Pool Metrics
        WorkersActive prometheus.Gauge
        WorkQueueSize prometheus.Gauge
        WorkItemsProcessed prometheus.Counter
        WorkItemsErrored prometheus.Counter
        WorkerProcessingTime prometheus.Histogram

        // Storage Metrics
        StorageOperations *prometheus.CounterVec
        StorageErrors *prometheus.CounterVec
        StorageSize prometheus.Gauge
        QueryTime prometheus.Histogram

        // API Metrics
        APIRequestsTotal *prometheus.CounterVec
        APIRequestDuration *prometheus.HistogramVec
        APIErrors *prometheus.CounterVec
}

var (
        metrics *Metrics
        once sync.Once
)

// GetMetrics returns the singleton metrics instance
func GetMetrics() *Metrics {
        once.Do(func() {
                metrics = initMetrics()
                // The promauto factory automatically registers metrics with the default registry
                // so we don't need to register them again with prometheus.MustRegister
        })
        return metrics
}

func initMetrics() *Metrics {
        return &Metrics{
                // Log Processing Metrics
                LogBatchesReceived: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_log_batches_received_total",
                        Help: "The total number of log batches received",
                }),
                LogEntriesReceived: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_log_entries_received_total",
                        Help: "The total number of log entries received",
                }),
                LogEntriesProcessed: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_log_entries_processed_total",
                        Help: "The total number of log entries successfully processed",
                }),
                LogEntriesFiltered: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_log_entries_filtered_total",
                        Help: "The total number of log entries filtered out",
                }),
                LogEntriesErrored: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_log_entries_errored_total",
                        Help: "The total number of log entries that had processing errors",
                }),
                LogProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
                        Name: "logstream_log_processing_duration_seconds",
                        Help: "The time taken to process a log entry",
                        Buckets: prometheus.DefBuckets,
                }),

                // Worker Pool Metrics
                WorkersActive: promauto.NewGauge(prometheus.GaugeOpts{
                        Name: "logstream_workers_active",
                        Help: "The number of active workers in the pool",
                }),
                WorkQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
                        Name: "logstream_work_queue_size",
                        Help: "The current size of the work queue",
                }),
                WorkItemsProcessed: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_work_items_processed_total",
                        Help: "The total number of work items processed",
                }),
                WorkItemsErrored: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "logstream_work_items_errored_total",
                        Help: "The total number of work items that resulted in errors",
                }),
                WorkerProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
                        Name: "logstream_worker_processing_duration_seconds",
                        Help: "The time taken by a worker to process a work item",
                        Buckets: prometheus.DefBuckets,
                }),

                // Storage Metrics
                StorageOperations: promauto.NewCounterVec(
                        prometheus.CounterOpts{
                                Name: "logstream_storage_operations_total",
                                Help: "The total number of storage operations by type",
                        },
                        []string{"operation"},
                ),
                StorageErrors: promauto.NewCounterVec(
                        prometheus.CounterOpts{
                                Name: "logstream_storage_errors_total",
                                Help: "The total number of storage errors by type",
                        },
                        []string{"operation"},
                ),
                StorageSize: promauto.NewGauge(prometheus.GaugeOpts{
                        Name: "logstream_storage_size_bytes",
                        Help: "The current size of the storage in bytes",
                }),
                QueryTime: promauto.NewHistogram(prometheus.HistogramOpts{
                        Name: "logstream_query_duration_seconds",
                        Help: "The time taken to execute a query",
                        Buckets: prometheus.DefBuckets,
                }),

                // API Metrics
                APIRequestsTotal: promauto.NewCounterVec(
                        prometheus.CounterOpts{
                                Name: "logstream_api_requests_total",
                                Help: "The total number of API requests",
                        },
                        []string{"method", "path", "status"},
                ),
                APIRequestDuration: promauto.NewHistogramVec(
                        prometheus.HistogramOpts{
                                Name: "logstream_api_request_duration_seconds",
                                Help: "The duration of API requests",
                                Buckets: prometheus.DefBuckets,
                        },
                        []string{"method", "path"},
                ),
                APIErrors: promauto.NewCounterVec(
                        prometheus.CounterOpts{
                                Name: "logstream_api_errors_total",
                                Help: "The total number of API errors",
                        },
                        []string{"method", "path", "error_type"},
                ),
        }
}
