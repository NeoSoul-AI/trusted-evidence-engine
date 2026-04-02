package metrics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const serviceName = "trusted-evidence-engine"

var (
	jobRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lifefun_job_runs_total",
			Help: "Total job runs partitioned by service, job, and result.",
		},
		[]string{"service", "job", "result"},
	)
	jobLastSuccessUnix = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lifefun_job_last_success_unixtime",
			Help: "Unix timestamp of the last successful job run partitioned by service and job.",
		},
		[]string{"service", "job"},
	)
	jobDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lifefun_job_duration_seconds",
			Help:    "Job duration partitioned by service, job, and result.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "job", "result"},
	)
	dependencyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lifefun_dependency_requests_total",
			Help: "Total dependency requests partitioned by service, dependency, and operation.",
		},
		[]string{"service", "dependency", "operation"},
	)
	dependencyFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lifefun_dependency_failures_total",
			Help: "Total dependency failures partitioned by service, dependency, operation, and kind.",
		},
		[]string{"service", "dependency", "operation", "kind"},
	)
	dependencyDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lifefun_dependency_duration_seconds",
			Help:    "Dependency request duration partitioned by service, dependency, and operation.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "dependency", "operation"},
	)
	teeHTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lifefun_tee_http_requests_total",
			Help: "Total trusted-evidence-engine HTTP requests partitioned by path, method, and status code.",
		},
		[]string{"path", "method", "status_code", "status_class"},
	)
	teeHTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lifefun_tee_http_request_duration_seconds",
			Help:    "Trusted-evidence-engine HTTP request duration partitioned by path, method, and status class.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status_class"},
	)
	teeResolveItemsReturned = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lifefun_tee_resolve_items_returned",
			Help: "Number of evidence items returned by the latest resolve request.",
		},
	)
	teeProviderDocumentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lifefun_tee_provider_documents_total",
			Help: "Total source documents returned by each trusted-evidence-engine provider.",
		},
		[]string{"provider"},
	)
)

func ObserveHTTP(path, method string, statusCode int, duration time.Duration) {
	statusClass := statusClass(statusCode)
	teeHTTPRequestsTotal.WithLabelValues(normalize(path), normalize(method), normalizeStatusCode(statusCode), statusClass).Inc()
	teeHTTPRequestDurationSeconds.WithLabelValues(normalize(path), normalize(method), statusClass).Observe(duration.Seconds())
}

func ObserveResolve(result string, duration time.Duration, itemCount int) {
	result = normalizeResult(result)
	jobRunsTotal.WithLabelValues(serviceName, "resolve", result).Inc()
	jobDurationSeconds.WithLabelValues(serviceName, "resolve", result).Observe(duration.Seconds())
	if result == "success" {
		jobLastSuccessUnix.WithLabelValues(serviceName, "resolve").Set(float64(time.Now().Unix()))
	}
	teeResolveItemsReturned.Set(float64(itemCount))
}

func ObserveProvider(provider string, startedAt time.Time, err error, documentCount int) {
	provider = normalize(provider)
	dependencyRequestsTotal.WithLabelValues(serviceName, provider, "fetch").Inc()
	dependencyDurationSeconds.WithLabelValues(serviceName, provider, "fetch").Observe(time.Since(startedAt).Seconds())
	if err != nil {
		dependencyFailuresTotal.WithLabelValues(serviceName, provider, "fetch", classifyError(err)).Inc()
		return
	}
	if documentCount > 0 {
		teeProviderDocumentsTotal.WithLabelValues(provider).Add(float64(documentCount))
	}
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func normalizeResult(result string) string {
	result = normalize(result)
	if result == "ok" {
		return "success"
	}
	return result
}

func normalizeStatusCode(statusCode int) string {
	if statusCode <= 0 {
		return "000"
	}
	return fmt.Sprintf("%d", statusCode)
}

func statusClass(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "5xx"
	case statusCode >= 400:
		return "4xx"
	case statusCode >= 300:
		return "3xx"
	case statusCode >= 200:
		return "2xx"
	default:
		return "unknown"
	}
}

func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(text, "429"), strings.Contains(text, "rate limit"), strings.Contains(text, "too many requests"):
		return "rate_limit"
	case strings.Contains(text, "timeout"), strings.Contains(text, "deadline exceeded"), strings.Contains(text, "timed out"), strings.Contains(text, "i/o timeout"):
		return "timeout"
	case strings.Contains(text, "401"), strings.Contains(text, "403"), strings.Contains(text, "unauthorized"), strings.Contains(text, "forbidden"):
		return "unauthorized"
	case strings.Contains(text, "400"), strings.Contains(text, "invalid"), strings.Contains(text, "bad request"):
		return "invalid_request"
	case strings.Contains(text, "500"), strings.Contains(text, "502"), strings.Contains(text, "503"), strings.Contains(text, "504"), strings.Contains(text, "server error"):
		return "server_error"
	case strings.Contains(text, "dial tcp"), strings.Contains(text, "connection refused"), strings.Contains(text, "connection reset"), strings.Contains(text, "no such host"), strings.Contains(text, "network"):
		return "network"
	default:
		return "unknown"
	}
}
