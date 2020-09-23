package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// LogMessageCount store all processed log messages per one container
var LogMessageCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "log_message_count",
		Help: "Store all processed log messages per one container",
	}, []string{"namespace", "pod_name", "container_name"})

func init() {
	prometheus.MustRegister(LogMessageCount)
}

// ServeHTTPRequests start http service for handle metrics
func ServeHTTPRequests(addr string, path string) {
	log.Printf("Start serving metrics on '%s/%s'", addr, path)
	http.Handle(path, promhttp.Handler())
	log.Fatal(http.ListenAndServe(addr, nil))
}
