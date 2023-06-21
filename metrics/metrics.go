package metrics

import (
	"fmt"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	DistanceFromFinalizedSlot = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "oracle",
			Name:      "distance_from_finalized_slot",
			Help:      "Distance from the latest finalized slot in slots",
		},
	)

	LatestProcessedSlot = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "oracle",
			Name:      "latest_processed_slot",
			Help:      "Latest processed slot by the oracle",
		},
	)

	LatestProcessedBlock = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "oracle",
			Name:      "latest_processed_block",
			Help:      "Latest processed block by the oracle",
		},
	)

	KnownRootAndSlot = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "oracle",
			Name:      "known_root_and_slot",
			Help:      "Known merkle root and the slot it belongs",
		},
		[]string{
			"slot",
			"merkle_root",
		},
	)

	Version = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "oracle",
			Name:      "version",
			Help:      "Oracle version",
		},
		[]string{
			"version",
		},
	)

	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "oracle",
			Name:      "http_requests_total",
			Help:      "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
		},
		[]string{"code", "method", "path"},
	)

	HttpRequestsLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "oracle",
			Name:      "http_request_duration_seconds",
			Help:      "How long it took to process the request, partitioned by status code, method and HTTP path.",
			Buckets:   []float64{0.3, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"code", "method", "path"},
	)
)

func RunMetrics(port int) {
	go func() {
		log.Info("Prometheus server started on port: ", port)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}()
}
